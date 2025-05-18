package main

import (
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/interfaces"
	apis "github.com/rpsoftech/golang-servers/servers/jwelly/main-server/api"
	"github.com/rpsoftech/golang-servers/utility/mongodb"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func deferMainFunc() {
	println("Closing...")
	mongodb.DeferFunction()
	// redis.DeferFunction()
}

func main() {
	defer deferMainFunc()

	app := fiber.New(fiber.Config{
		ServerHeader: "Bullion Server V1.0.0",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			mappedError, ok := err.(*interfaces.RequestError)
			if !ok {
				println(err.Error())
				return c.Status(500).JSON(interfaces.RequestError{
					Code:    interfaces.ERROR_INTERNAL_SERVER,
					Message: "Some Internal Error",
					Name:    "Global Error Handler Function",
				})
			}
			if mappedError.LogTheDetails {
				//Todo: Store The Details Of the Error With Body And Other Extra Details Like AUTH KEY AND ETC
			}
			return c.Status(mappedError.StatusCode).JSON(mappedError)
		},
	})
	// TODO Add middleware to recover from panics https://docs.gofiber.io/api/middleware/recover
	app.Use(logger.New())
	// app.Use(bullion_main_server_middleware.TokenDecrypter)

	if env.Env.APP_ENV != env.APP_ENV_PRODUCTION {

		// app.Get("/token", func(c *fiber.Ctx) error {
		// 	a, _ := bullion_main_server_services.AccessTokenService.GenerateToken(bullion_main_server_interfaces.GeneralUserAccessRefreshToken{
		// 		Role: bullion_main_server_interfaces.ROLE_ADMIN,
		// 		GeneralPurposeTokenGeneration: &jwt.GeneralPurposeTokenGeneration{
		// 			RegisteredClaims: &j.RegisteredClaims{
		// 				IssuedAt: j.NewNumericDate(time.Now()),
		// 			},
		// 		},
		// 	})
		// 	return c.SendString(a)
		// }).Name("Temp Admin Access Token")
	}
	// bullion_main_server_repos.BullionSiteInfoRepo.Save(bullion_main_server_interfaces.CreateNewBullionSiteInfo("Akshat Bullion", "https://akshatbullion.com").AddGeneralUserInfo(true, true))
	// app.Get("/", func(c *fiber.Ctx) error {
	// bull := bullion_main_server_repos.BullionSiteInfoRepo.FindOne("ad3cee16-e8d7-4a27-a060-46d99c133273")
	// return c.JSON(bull)

	// return c.JSON(bullion_main_server_repos.BullionSiteInfoRepo.FindOneByDomain("https://akshatgold.com"))
	// return c.JSON(bullion_main_server_repos.BullionSiteInfoRepo.FindOneByDomain("https://akshatbullion.com"))
	// return c.SendString("Hello, World!")
	// })
	// bullion_main_server_apis.AddApis(app.Group("/v1"))
	apis.AddApis(app.Group("/v1"))
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Sorry can't find that!")
	})
	hostAndPort := ""
	if env.Env.APP_ENV == env.APP_ENV_LOCAL || env.Env.APP_ENV == env.APP_ENV_DEVELOPE {
		hostAndPort = "127.0.0.1"
	}
	// Print the router stack in JSON format
	// data, _ := json.MarshalIndent(app.Stack(), "", "  ")
	// fmt.Println(string(data))

	hostAndPort = hostAndPort + ":" + env.GetServerPort(env.PORT_KEY)
	app.Listen(hostAndPort)
	// log.Fatal(app.Listen(":" + strconv.Itoa(env.Env.PORT)))
}
