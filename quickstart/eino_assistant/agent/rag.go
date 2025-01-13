package agent

// func RunTestRAG() {
// 	ctx := context.Background()

// 	embed, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
// 		Model:   os.Getenv("OPENAI_EMBEDDING_MODEL"),
// 		ByAzure: true,
// 		APIKey:  os.Getenv("OPENAI_API_KEY"),
// 		BaseURL: os.Getenv("OPENAI_BASE_URL"),
// 	})
// 	if err != nil {
// 		panic(err)
// 	}

// 	redisStore, err := redis.NewRedisVectorStore(ctx, &redis.RedisVectorStoreConfig{
// 		RedisAddr:      "127.0.0.1:6379",
// 		Embedding:      embed,
// 		RedisKeyPrefix: "vector:",
// 		Dimension:      3072,
// 		TopK:           3,
// 	})
// 	if err != nil {
// 		panic(err)
// 	}

// 	docIDs, err := redisStore.Store(ctx, []*schema.Document{
// 		{
// 			Content: "Andy: Hello, world! this is Andy",
// 			MetaData: map[string]interface{}{
// 				"name": "Andy",
// 			},
// 		},
// 		{
// 			Content: "Peter: Hello, world! this is Peter",
// 			MetaData: map[string]interface{}{
// 				"name": "Peter",
// 			},
// 		},
// 		{
// 			Content: "Tom: how are you?",
// 			MetaData: map[string]interface{}{
// 				"name": "Tom",
// 			},
// 		},
// 		{
// 			Content: "Tom: I am fine",
// 			MetaData: map[string]interface{}{
// 				"name": "Tom",
// 			},
// 		},
// 	})
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println(docIDs)

// 	xdocs, err := redisStore.Retrieve(ctx, "what does Tom say?")
// 	if err != nil {
// 		panic(err)
// 	}

// 	for i, doc := range xdocs {
// 		fmt.Printf("doc %d: %s\n", i, doc.Content)
// 		fmt.Printf("doc %d: %v\n", i, doc.MetaData)
// 	}

// }

// func Retriever(ctx context.Context, embedding embedding.Embedder) retriever.Retriever {
// 	redisRetriever, err := redis.NewRedisVectorStore(ctx, &redis.RedisVectorStoreConfig{
// 		RedisAddr:      "127.0.0.1:6379",
// 		RedisKeyPrefix: "eino:markdown",
// 		Dimension:      1536,
// 		TopK:           3,
// 		MinScore:       0.5,
// 		Embedding:      embedding,
// 	})
// 	if err != nil {
// 		panic(err)
// 	}

// 	return redisRetriever
// }
