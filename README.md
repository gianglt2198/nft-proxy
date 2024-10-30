# nft-proxy
NFT Image proxy to speed up retrieval &amp; reduce reliance on RPC


1. Caches all NFTs in smaller 500x500 size
2. Provides direct mint -> image REST API
3. Provides image resizing on the fly (stretch)

## Bugs Report 

### Context timeout
- Almost functions don't have context timeout as param
- As Database level, it is vulnerable to lock database

### Database 
- Sometimes, the connections are blocked from somewhere 
- Add Retry Backoff Strategy 

### Solana Rate Limit 
- We need to consider solana ratelimit
- Add ratelimit to api layer or solana service

## Improvements/Optimizations Report 

### Open connection poolings 
```
sqlDB, err := sql.Open("sqlite3", ds.database)
if err != nil {
    return fmt.Errorf("failed to open database: %w", err)
}

// Configure connection pool
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)
```

### Support multiple nft requests
- Perhaps, check the rate limit of solana's rpc
- Apply concurrent processing and Use transaction for multiple creations
- Or Use create by batch

### Handle Image
-  If image is over size, process image by chunks

### File store in cloud or file system
- Mount directory cache to cloud or file system service