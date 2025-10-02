package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"ssot/gql/graphql/graph"
	"ssot/gql/graphql/internal/auth"
	"ssot/gql/graphql/internal/services"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	awscfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	dynamoClient := dynamodb.NewFromConfig(awscfg)

	serviceConfig := services.LoadServiceConfigFromEnv(dynamoClient)
	serviceManager := services.NewServiceManager(serviceConfig)

	resolver := &graph.Resolver{
		ServiceManager: serviceManager,
	}

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	// Setup routes
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Authentication endpoints (no auth required)
	// mux.HandleFunc("/auth/login", auth.LoginHandler)
	// mux.HandleFunc("/auth/register", auth.CreateUserHandler)

	// GraphQL playground
	mux.Handle("/", playground.Handler("GraphQL playground", "/query"))

	// GraphQL endpoint with authentication middleware
	mux.Handle("/query", auth.Middleware(srv))

	log.Printf("starting the server at :%s for GraphQL", port)
	log.Printf("using DynamoDB loan cash flow table: %s", serviceConfig.LoanCashFlowTableName)
	// log.Printf("Authentication endpoints:")
	// log.Printf("  POST /auth/login - Login with email/password")
	// log.Printf("  POST /auth/register - Register new user")
	log.Printf("  GET / - GraphQL playground")
	log.Printf("  POST /query - GraphQL API (requires JWT token)")
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
