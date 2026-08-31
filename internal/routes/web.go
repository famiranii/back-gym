package routes

import (
	"github.com/famiranii/back-gym.git/internal/api"
	"github.com/famiranii/back-gym.git/internal/api/handlers"
	"github.com/famiranii/back-gym.git/internal/api/middleware"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/static"
)

func SetupRoutes(server *api.Server) error {
	server.App.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3001"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	auth := middleware.AuthMiddleware(server.TokenMaker)

	// Auth
	user := handlers.NewUserHandler(server.Store, server.TokenMaker, server.Config)
	server.App.Post("/register", user.RegisterUser)
	server.App.Post("/login", user.LoginUser)
	server.App.Get("/users", user.GetAllUsers)
	server.App.Get("/users/:id", auth, user.GetUser)

	// Products
	product := handlers.NewProductHandler(server.Store, server.TokenMaker, server.Config)
	server.App.Get("/products", product.GetAllProducts)
	server.App.Get("/products/:id", product.GetProduct)
	server.App.Post("/products", auth, product.CreateProduct)
	server.App.Put("/products/:id", auth, product.UpdateProduct)
	server.App.Delete("/products/:id", auth, product.DeleteProduct)

	// Variants
	variant := handlers.NewVariantHandler(server.Store, server.TokenMaker, server.Config)
	server.App.Get("/products/:id/variants", variant.GetVariantsByProduct)
	server.App.Post("/products/:id/variants", auth, variant.CreateVariant)
	server.App.Put("/variants/:id", auth, variant.UpdateVariantStock)
	server.App.Delete("/variants/:id", auth, variant.DeleteVariant)

	//category
	category := handlers.NewCategoryHandler(server.Store, server.TokenMaker, server.Config)

	server.App.Get("/categories", category.GetAllCategories)
	server.App.Get("/categories/:id", category.GetCategoryByID)
	server.App.Post("/categories", auth, category.CreateCategory)
	server.App.Delete("/categories/:id", auth, category.DeleteCategory)

	//upload images
	server.App.Post("/upload", auth, handlers.UploadImage)
	server.App.Get("/uploads/*", static.New("./uploads"))

	// Reviews
	review := handlers.NewReviewHandler(server.Store)
	server.App.Post("/products/:id/reviews", review.UpsertReview)
	server.App.Get("/products/:id/reviews", review.GetReviews)
	server.App.Delete("/products/:id/reviews", review.DeleteReview)

	// Addresses
	address := handlers.NewAddressHandler(server.Store)
	server.App.Get("/users/:id/addresses", auth, address.GetAddresses)
	server.App.Post("/users/:id/addresses", auth, address.CreateAddress)
	server.App.Put("/users/:id/addresses/:addr_id", auth, address.UpdateAddress)
	server.App.Delete("/users/:id/addresses/:addr_id", auth, address.DeleteAddress)
	server.App.Patch("/users/:id/addresses/:addr_id/default", auth, address.SetDefault)

	// Cart
	cart := handlers.NewCartHandler(server.Store)
	server.App.Get("/cart", auth, cart.GetCart)
	server.App.Post("/cart", auth, cart.AddToCart)
	server.App.Patch("/cart/:id", auth, cart.UpdateCartItem)
	server.App.Delete("/cart/:id", auth, cart.RemoveFromCart)
	server.App.Delete("/cart", auth, cart.ClearCart)
	return nil
}
