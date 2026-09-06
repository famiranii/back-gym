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
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowCredentials: true,
	}))

	auth := middleware.AuthMiddleware(server.TokenMaker)

	// Auth
	user := handlers.NewUserHandler(server.Store, server.TokenMaker, server.Config)
	server.App.Post("/register", user.RegisterUser) // done
	server.App.Post("/login", user.LoginUser)       // done
	server.App.Get("/users", user.GetAllUsers)
	server.App.Get("/users/me", auth, user.GetMe)
	server.App.Get("/users/:id", auth, user.GetUser)

	// Products
	product := handlers.NewProductHandler(server.Store)
	server.App.Get("/products", product.GetAllProducts)             // done
	server.App.Get("/products/:id", product.GetProduct)             //done
	server.App.Post("/products", auth, product.CreateProduct)       // done
	server.App.Put("/products/:id", auth, product.UpdateProduct)    //done
	server.App.Delete("/products/:id", auth, product.DeleteProduct) //done

	// Variants
	variant := handlers.NewVariantHandler(server.Store)
	server.App.Get("/products/:id/variants", variant.GetVariantsByProduct)
	server.App.Post("/products/:id/variants", auth, variant.CreateVariant)
	server.App.Put("/variants/:id", auth, variant.UpdateVariantStock)
	server.App.Delete("/variants/:id", auth, variant.DeleteVariant)

	//category
	category := handlers.NewCategoryHandler(server.Store)

	server.App.Get("/categories", category.GetAllCategories)            //done
	server.App.Get("/categories/:id", category.GetCategoryByID)         // done
	server.App.Post("/categories", auth, category.CreateCategory)       // done
	server.App.Delete("/categories/:id", auth, category.DeleteCategory) //done

	//upload images
	server.App.Post("/upload", auth, handlers.UploadImage)
	server.App.Get("/uploads/*", static.New("./uploads"))

	// Reviews
	review := handlers.NewReviewHandler(server.Store)
	server.App.Post("/products/:id/reviews", auth, review.UpsertReview)   // done
	server.App.Get("/products/:id/reviews", review.GetReviews)            // done
	server.App.Delete("/products/:id/reviews", auth, review.DeleteReview) // done

	// Addresses
	address := handlers.NewAddressHandler(server.Store)
	server.App.Get("/users/:id/addresses", auth, address.GetAddresses)              //done
	server.App.Get("/users/:id/addresses/:addr_id", auth, address.GetAddress)       //done
	server.App.Post("/users/:id/addresses", auth, address.CreateAddress)            //done
	server.App.Put("/users/:id/addresses/:addr_id", auth, address.UpdateAddress)    // done
	server.App.Delete("/users/:id/addresses/:addr_id", auth, address.DeleteAddress) // done
	server.App.Patch("/users/:id/addresses/:addr_id/default", auth, address.SetDefault)

	// Cart
	cart := handlers.NewCartHandler(server.Store)
	server.App.Get("/cart", auth, cart.GetCart)               //done
	server.App.Post("/cart", auth, cart.AddToCart)            //done
	server.App.Patch("/cart/:id", auth, cart.UpdateCartItem)  //done
	server.App.Delete("/cart/:id", auth, cart.RemoveFromCart) //done
	server.App.Delete("/cart", auth, cart.ClearCart)          // done

	// Shipping
	shipping := handlers.NewShippingHandler(server.Store)
	server.App.Get("/shipping-cost", shipping.GetShippingCost)
	server.App.Put("/admin/shipping-cost", auth, shipping.UpdateShippingCost)

	// Orders
	order := handlers.NewOrderHandler(server.Store)
	server.App.Post("/orders", auth, order.CreateOrder)
	server.App.Get("/orders", auth, order.GetMyOrders)
	server.App.Get("/orders/:id", auth, order.GetOrderDetail)
	server.App.Patch("/orders/:id/status", auth, order.UpdateOrderStatus)
	return nil
}
