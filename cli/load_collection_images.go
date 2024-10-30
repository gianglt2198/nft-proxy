package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	nft_proxy "github.com/alphabatem/nft-proxy"
	services "github.com/alphabatem/nft-proxy/service"
	"github.com/babilu-online/common/context"
	token_metadata "github.com/gagliardetto/metaplex-go/clients/token-metadata"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/joho/godotenv"
	"gorm.io/gorm/clause"
)

type collectionLoader struct {
	metaWorkerCount  int
	fileWorkerCount  int
	mediaWorkerCount int

	metaDataIn chan *token_metadata.Metadata
	fileDataIn chan *nft_proxy.NFTMetadataSimple
	mediaIn    chan *nft_proxy.Media
}

const (
	IMAGE_PATH  = "./images"
	DefaultSize = 500
)

func int() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {

	mainContext, err := context.NewCtx(
		&services.SqliteService{},
		&services.ResizeService{},
	)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = mainContext.Run()
	if err != nil {
		panic(err)
	}

	err = run(mainContext)
	if err != nil {
		panic(err)
	}
}

func run(ctx *context.Context) error {
	log.Printf("Loading collection images: %s", "TODO")

	l := collectionLoader{
		metaWorkerCount:  3,
		fileWorkerCount:  3,
		mediaWorkerCount: 1,
		metaDataIn:       make(chan *token_metadata.Metadata),
		fileDataIn:       make(chan *nft_proxy.NFTMetadataSimple),
		mediaIn:          make(chan *nft_proxy.Media),
	}

	l.spawnWorkers(ctx)

	//Load collection
	err := l.loadCollection(ctx)
	if err != nil {
		panic(err)
	}

	return nil
}

func (l *collectionLoader) spawnWorkers(ctx *context.Context) {
	db := ctx.Service(services.SQLITE_SVC).(*services.SqliteService)
	resize := ctx.Service(services.RESIZE_SVC).(*services.ResizeService)
	httpClient := &http.Client{Timeout: 5 * time.Second}

	for i := 0; i < l.metaWorkerCount; i++ {
		go l.metaDataWorker()
	}
	for i := 0; i < l.fileWorkerCount; i++ {
		go l.fileDataWorker(httpClient, resize)
	}
	for i := 0; i < l.mediaWorkerCount; i++ {
		go l.mediaWorker(db)
	}
}

func (l *collectionLoader) loadCollection(ctx *context.Context) error {
	// Default Config of Collection ID/Address: TODO
	collectionAddress := "09b36de0a9862cdbfe3eb023959e97e816e17d846a5762b3c1bfe7b75490c23b"
	collectionPubkey, err := solana.PublicKeyFromBase58(collectionAddress)
	//TODO Fetch all the mints for that collection
	//TODO Fetch Mints/Hash List

	//TODO Fetch all the metadata accounts for that collection
	//TODO Fetch all images for the accounts
	client := rpc.New(os.Getenv("RPC_URL"))
	accounts, err := client.GetProgramAccountsWithOpts(
		context.Background(),
		collectionPubkey,
		&rpc.GetProgramAccountsOpts{
			Filters: []rpc.RPCFilter{
				{
					MemCmp: &rpc.RPCFilterMemCmp{
						Offset: 326, // Offset for collection address in metadata account
						Bytes:  collectionPubkey.Bytes(),
					},
				},
			},
		},
	)
	if err != nil {
		log.Printf("%v - %v", collectionPubkey, err)
		return err
	}

	//TODO Batch into batches of 100
	//TODO Pass to metaDataIn<-
	batchSize := 100
	batches := getBatches(len(accounts), batchSize)
	wg := sync.WaitGroup{}
	wg.Add(len(batches))
	for _, batch := range batches {
		start, end := batch[0], batch[1]
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				account := accounts[i]
				var metadata *token_metadata.Metadata
				err := metadata.UnmarshalBorsh(account.Account.Data)
				if err != nil {
					log.Printf("Warning: failed to decode metadata for account %s: %v", account.Pubkey, err)
					continue
				}
				if metadata.Collection != nil && metadata.Collection.Key.Equals(collectionPubkey) {
					l.metaDataIn <- metadata
				}
			}
		}(start, end)
	}

	wg.Wait()
	close(l.metaDataIn)
	return nil
}

// Fetches the off-chain data from the on-chain account & passes to `fileDataWorker`
func (l *collectionLoader) metaDataWorker() {
	client := &http.Client{Timeout: 5 * time.Second}
	for m := range l.metaDataIn {
		//TODO Fetch Off-Chain Data
		//TODO Pass to fileDataIn<-
		metadata, err := fetchOffChainData(m.Data.Uri, client)
		if err != nil {
			log.Printf("Error fetching off-chain data for %s: %v", m.Data.Uri, err)
			continue
		}

		// Add mint address to metadata
		metadata.Mint = m.Mint.String()

		l.fileDataIn <- metadata
	}
}

// Downloads required files & passes to `mediaWorker`
func (l *collectionLoader) fileDataWorker(httpClient *http.Client, resize *services.ResizeService) {
	//TODO Fetch Image
	//TODO Resize Image 500x500
	//TODO Fetch Media
	for f := range l.fileDataIn {
		media := &nft_proxy.Media{
			Mint:            f.Mint,
			Name:            f.Name,
			Symbol:          f.Symbol,
			MintDecimals:    f.Decimals,
			UpdateAuthority: f.UpdateAuthority,
			ImageUri:        f.Image,
		}

		if f.Image != "" {
			extension := strings.ToLower(filepath.Ext(f.Image))
			switch extension {
			case ".png":
				media.ImageType = "png"
			case ".jpg", ".jpeg":
				media.ImageType = "jpeg"
			case ".gif":
				media.ImageType = "gif"
			default:
				media.ImageType = "jpeg" // Default type
			}
		}

		// Handle animation URL if present
		mediaFile := f.AnimationFile()
		if mediaFile != nil {
			media.MediaUri = mediaFile.URL
			mediaFile.Type = "mp4"
			if strings.Contains(mediaFile.Type, "/") {
				media.MediaType = strings.Split(mediaFile.Type, "/")[1]
			}
		}

		// Check Files array if no animation URL was found
		if media.MediaUri == "" && len(f.Files) > 0 {
			for _, file := range f.Files {
				// Look for video files in the Files array
				fileType := strings.ToLower(file.Type)
				if strings.Contains(fileType, "video") {
					media.MediaUri = file.URL
					if strings.Contains(fileType, "/") {
						media.MediaType = strings.Split(fileType, "/")[1]
					}
					break
				}
			}
		}

		// Process image if ImageUri exists
		if media.ImageUri != "" {
			// Create directory if it doesn't exist
			if err := os.MkdirAll(IMAGE_PATH, 0755); err != nil {
				log.Printf("Error creating directory: %v", err)
			}

			// Generate local path for the image
			media.LocalPath = filepath.Join(IMAGE_PATH, fmt.Sprintf("%s.%s", media.Mint, media.ImageType))

			// Download and resize image
			err := fetchAndResizeImage(media.ImageUri, media.LocalPath, httpClient, resize)
			if err != nil {
				log.Printf("Error processing image for %s: %v", media.Mint, err)
			}
		}

		// Pass to next worker
		l.mediaIn <- media
	}
}

// Stores media data down to SQL
func (l *collectionLoader) mediaWorker(db *services.SqliteService) {
	for m := range l.mediaIn {
		//TODO SAVE TO DB
		log.Printf(db.Db().Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "mint"}}, // key colum
			UpdateAll: true,
		}).Create(&m).Error)
	}
}

func getBatches(length, batchSize int) [][2]int {
	var batches [][2]int
	for start := 0; start < length; start += batchSize {
		end := start + batchSize
		if end > length {
			end = length
		}
		batches = append(batches, [2]int{start, end})
	}
	return batches
}

func fetchOffChainData(uri string, client *http.Client) (*nft_proxy.NFTMetadataSimple, error) {
	resp, err := client.Get(strings.Trim(uri, "\x00"))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data nft_proxy.NFTMetadataSimple
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func fetchAndResizeImage(uri string, store string, client *http.Client, resize *services.ResizeService) error {
	if uri == "" {
		return errors.New("invalid image")
	}

	var err error
	var data []byte
	if strings.Contains(uri, nft_proxy.BASE64_PREFIX) {
		base64String := uri
		// Remove the data:image/jpeg;base64, prefix if present
		if v := strings.Index(base64String, nft_proxy.BASE64_PREFIX); v > -1 {
			base64String = base64String[v+len(nft_proxy.BASE64_PREFIX):]
		}

		data, err = base64.StdEncoding.DecodeString(base64String)
		if err != nil {
			return err
		}
	} else {
		uri = strings.Replace(strings.TrimSpace(uri), ".ipfs.nftstorage.link", ".ipfs.w3s.link", 1)

		log.Println("Fetching", uri)

		req, err := http.NewRequest("GET", uri, nil)
		if err != nil {
			return err
		}

		req.Header.Set("User-Agent", "PostmanRuntime/7.29.2")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Encoding", "gzip,deflate,br")

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return errors.New(resp.Status)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	}

	if len(data) == 0 {
		return errors.New("failed to download image")
	}

	output, err := os.Create(store)
	if err != nil {
		return err
	}
	defer output.Close()

	err = resize.Resize(data, output, DefaultSize)
	if err != nil {
		return err
	}
	return nil
}
