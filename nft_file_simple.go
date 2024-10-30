package nft_proxy

import "strings"

type NFTMetadataSimple struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"-"`
	//Description          string               `json:"description"`
	//SellerFeeBasisPoints float64              `json:"seller_fee_basis_points"`
	Image        string `json:"image"`
	AnimationURL string `json:"animation_url"`
	ExternalURL  string `json:"external_url"`
	//Collection   NFTCollectionSimple `json:"collection"`
	Properties NFTPropertiesSimple `json:"properties"`
	//Attributes           []NFTAttributeSimple `json:"attributes"`
	Files []NFTFiles `json:"files"`

	UpdateAuthority string `json:"updateAuthority"`
	Mint            string `json:"mint"`
}

func (m *NFTMetadataSimple) AnimationFile() *NFTFiles {
	for _, f := range m.Files {
		if f.URL == m.Image || strings.Contains(f.Type, "image") && !strings.Contains(f.Type, "gif") {
			continue
		}
		return &f
	}
	return nil
}

func (m *NFTMetadataSimple) ImageFile() *NFTFiles {
	for _, f := range m.Files {
		if f.URL == m.Image {
			return &f
		}
	}

	return nil
}

type NFTFiles struct {
	URL  string `json:"URL"`
	Type string `json:"type"`
}

type NFTPropertiesSimple struct {
	Category string             `json:"category"`
	Creators []NFTCreatorSimple `json:"creators"`
}

type NFTCollectionSimple struct {
	Name   string `json:"name"`
	Family string `json:"family"`
}

type NFTCreatorSimple struct {
	Address string `json:"address"`
	//Verified bool   `json:"verified"`
}

type NFTAttributeSimple struct {
	TraitType string      `json:"trait_type"`
	Value     interface{} `json:"value"`
}

type NFTFileSimple struct {
	URI  string `json:"uri"`
	Type string `json:"type"`
}
