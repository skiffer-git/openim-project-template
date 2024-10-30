// main.go (Gin API Service)
package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net"
	"net/http"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"your_project_name/proto"
	"log"
	"os"
	"time"
)

var config Config

func main() {
	// Load configuration from environment variables
	config = Config{
		Version: os.Getenv("CONFIG_VERSION"),
	}

	deploymentType := os.Getenv("DEPLOYMENT_TYPE")
	grpcPort := os.Getenv("GRPC_PORT")
	var grpcAddress string
	if deploymentType == "kubernetes" {
		grpcAddress = fmt.Sprintf("grpc-service:%s", grpcPort)
	} else {
		grpcAddress = fmt.Sprintf("localhost:%s", grpcPort)
	}

	// Initialize Gin router
	r := gin.Default()

	r.POST("/set_username", func(c *gin.Context) {
		setUsernameHandler(c, grpcAddress)
	})
	r.GET("/get_usernames", func(c *gin.Context) {
		getUsernamesHandler(c, grpcAddress)
	})

	// Start HTTP server
	r.Run(":8080")
}

func setUsernameHandler(c *gin.Context, grpcAddress string) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Call gRPC service to save user
	conn, err := grpc.Dial(grpcAddress, grpc.WithInsecure())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to gRPC server"})
		return
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	_, err = client.SetUser(context.TODO(), &pb.SetUserRequest{Username: user.Username})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User saved successfully"})
}

func getUsernamesHandler(c *gin.Context, grpcAddress string) {
	// Call gRPC service to get users
	grpcAddress = fmt.Sprintf("grpc-service:%s", grpcPort)
	conn, err := grpc.Dial(grpcAddress, grpc.WithInsecure())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to gRPC server"})
		return
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	resp, err := client.GetUsers(context.TODO(), &pb.GetUsersRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  resp.Users,
		"config": config,
	})
}

// grpc_server.go (gRPC Service)
package main

import (
"context"
"fmt"
"net"
"google.golang.org/grpc"
"go.mongodb.org/mongo-driver/mongo"
"go.mongodb.org/mongo-driver/mongo/options"
"log"
"os"
"time"
pb "your_project_name/proto"
)

type server struct {
	pb.UnimplementedUserServiceServer
	usersCollection *mongo.Collection
}

func main() {
	// Load MongoDB configuration from environment variables
	deploymentType := os.Getenv("DEPLOYMENT_TYPE")
	var mongoURI string
	if deploymentType == "kubernetes" {
		mongoURI = fmt.Sprintf("mongodb://%s:%s@mongo-service:27017",
			os.Getenv("MONGO_USER"), os.Getenv("MONGO_PASSWORD"))
	} else {
		mongoURI = "mongodb://localhost:27017"
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		panic(err)
	}
	usersCollection := client.Database("userdb").Collection("users")

	// Start gRPC server
	deploymentType := os.Getenv("DEPLOYMENT_TYPE")
	grpcPort := os.Getenv("GRPC_PORT")
	if deploymentType != "kubernetes" || grpcPort == "" {
		grpcPort = "50051" // Default port for local source deployment
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &server{usersCollection: usersCollection})
	log.Printf("gRPC server listening on %v", listener.Addr())
	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *server) SetUser(ctx context.Context, in *pb.SetUserRequest) (*pb.SetUserResponse, error) {
	_, err := s.usersCollection.InsertOne(ctx, User{Username: in.Username})
	if err != nil {
		return nil, err
	}
	return &pb.SetUserResponse{}, nil
}

func (s *server) GetUsers(ctx context.Context, in *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {
	cursor, err := s.usersCollection.Find(ctx, mongo.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []string
	for cursor.Next(ctx) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user.Username)
	}
	return &pb.GetUsersResponse{Users: users}, nil
}