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
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://85.198.48.18:3000",
			"https://cheheltike.com",
			"https://www.cheheltike.com",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"DELETE",
			"OPTIONS",
			"PATCH",
		},
		AllowCredentials: true,
	}))

	optionalAuth := middleware.OptionalAuthMiddleware(server.TokenMaker)
	auth := middleware.AuthMiddleware(server.TokenMaker)
	adminAuth := middleware.AdminMiddleware(server.TokenMaker)

	// Auth
	user := handlers.NewUserHandler(server.Store, server.TokenMaker, server.Config)
	server.App.Post("/register", user.RegisterUser)
	server.App.Post("/login", user.LoginUser)
	server.App.Post("/logout", user.Logout)
	server.App.Post("/refresh", user.RefreshToken)

	// OTP: phone verification for register + password recovery
	otp := handlers.NewOTPHandler(server.Store, server.SMS, server.TokenMaker, server.Config)
	server.App.Post("/register/otp", otp.RequestRegisterOTP)
	server.App.Post("/register/verify", otp.VerifyRegisterOTP)
	server.App.Post("/password/forgot", otp.RequestPasswordReset)
	server.App.Post("/password/reset", otp.ResetPassword)
	server.App.Post("/otp/resend", otp.ResendOTP)
	server.App.Get("/users", adminAuth, user.GetAllUsers)
	server.App.Get("/users/me", auth, user.GetMe)
	server.App.Get("/users/search", adminAuth, user.SearchUsers) // ← قبل از :id
	server.App.Get("/users/:id", adminAuth, user.GetUser)
	server.App.Put("/users/me", auth, user.UpdateUser)

	// Products
	product := handlers.NewProductHandler(server.Store)

	server.App.Get("/products", product.GetAllProducts)
	server.App.Get("/products/search", product.SearchProducts)
	server.App.Get("/products/:id", optionalAuth, product.GetProduct)
	server.App.Get("/categories/:name/products", product.GetProductsByCategory)
	server.App.Post("/products", adminAuth, product.CreateProduct)
	server.App.Put("/products/:id", adminAuth, product.UpdateProduct)
	server.App.Delete("/products/:id", adminAuth, product.DeleteProduct)

	// Variants
	variant := handlers.NewVariantHandler(server.Store)
	server.App.Get("/products/:id/variants", adminAuth, variant.GetVariantsByProduct)
	server.App.Post("/products/:id/variants", adminAuth, variant.CreateVariant)
	server.App.Put("/variants/:id", adminAuth, variant.UpdateVariantStock)
	server.App.Delete("/variants/:id", adminAuth, variant.DeleteVariant)

	//category
	category := handlers.NewCategoryHandler(server.Store)

	server.App.Get("/categories", category.GetAllCategories)                 //done
	server.App.Get("/categories/:id", category.GetCategoryByID)              // done
	server.App.Post("/categories", adminAuth, category.CreateCategory)       // done
	server.App.Delete("/categories/:id", adminAuth, category.DeleteCategory) //done

	//upload images
	server.App.Post("/upload", adminAuth, handlers.UploadImage)
	server.App.Get("/uploads/*", static.New("./uploads"))

	// Reviews
	review := handlers.NewReviewHandler(server.Store)
	server.App.Post("/products/:id/reviews", auth, review.UpsertReview)
	server.App.Get("/products/:id/reviews", review.GetReviews)
	server.App.Get("/products/:id/reviews/rating", review.GetAverageRating)
	server.App.Delete("/products/:id/reviews", adminAuth, review.DeleteReview)
	server.App.Get("/admin/reviews/pending", adminAuth, review.GetPendingReviews)
	server.App.Patch("/admin/reviews/:id/approve", adminAuth, review.ApproveReview)
	server.App.Patch("/admin/reviews/:id/reject", adminAuth, review.RejectReview)
	server.App.Delete("/admin/reviews/:id", adminAuth, review.AdminDeleteReview)

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
	server.App.Get("/cart", optionalAuth, cart.GetCart)               //done
	server.App.Post("/cart", optionalAuth, cart.AddToCart)            //done
	server.App.Patch("/cart/:id", optionalAuth, cart.UpdateCartItem)  //done
	server.App.Delete("/cart/:id", optionalAuth, cart.RemoveFromCart) //done
	server.App.Delete("/cart", optionalAuth, cart.ClearCart)          // done

	// Shipping
	shipping := handlers.NewShippingHandler(server.Store)
	server.App.Get("/shipping-cost", shipping.GetShippingCost)
	server.App.Put("/admin/shipping-cost", adminAuth, shipping.UpdateShippingCost)

	// Orders
	order := handlers.NewOrderHandler(server.Store, server.SMS)
	server.App.Post("/orders", auth, order.CreateOrder)
	server.App.Get("/orders", auth, order.GetMyOrders)
	server.App.Get("/admin/orders/:id", adminAuth, order.GetUserOrders) // admin
	server.App.Get("/admin/orders/status/:status", adminAuth, order.GetAdminOrdersByStatus)
	server.App.Get("/orders/:id", auth, order.GetOrderDetail)
	server.App.Patch("/orders/:id/status", auth, order.UpdateOrderStatus)
	server.App.Get("/orders/status/:status", auth, order.GetOrdersByStatus)

	//discount
	discount := handlers.NewDiscountCodeHandler(server.Store)
	server.App.Post("/discount-codes", adminAuth, discount.Create)
	server.App.Get("/discount-codes", adminAuth, discount.List)
	server.App.Delete("/discount-codes/:id", adminAuth, discount.Delete)
	server.App.Post("/discount/validate", auth, discount.ValidateDiscount)

	//whish list
	whishList := handlers.NewWishlistHandler(server.Store)
	server.App.Get("/wishlist", auth, whishList.GetWishlist)
	server.App.Post("/wishlist", auth, whishList.ToggleWishlist)
	server.App.Delete("wishlist/:product_id", auth, whishList.RemoveFromWishlist)

	//dashboard
	admin := handlers.NewAdminHandler(server.Store)

	server.App.Get("/admin/dashboard", adminAuth, admin.GetDashboard)

	// SMS
	smsHandler := handlers.NewSMSHandler(server.SMS)
	server.App.Post("/admin/sms", adminAuth, smsHandler.SendSMS)

	banner := handlers.NewBannerHandler(server.Store, server.TokenMaker, server.Config)

	server.App.Get("/banners", banner.GetActiveBanners)
	server.App.Post("/banners", auth, banner.CreateBanner)
	server.App.Put("/banners/:id", auth, banner.UpdateBanner)
	server.App.Delete("/banners/:id", auth, banner.DeleteBanner)

	placeHandler := handlers.NewPlaceHandler(
		server.Config.MAPIR_API_KEY,
		server.Config.MAPIR_DAILY_REQUEST_LIMIT,
	)

	server.App.Get("/places/search", placeHandler.Search)
	return nil
}
