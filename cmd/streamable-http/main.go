package streamable_http

import (
	"os"
)

func main() {
	// initialize postgres database with postgres://user:password@localhost:5432/launchpad?sslmode=disable
	_ = os.Getenv("POSTGRES_URL")
	//db, err := gorm.Open(postgres.Open(postgresUrl), &gorm.Config{})
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//// Initialize services and hooks
	//evmService, txService, uniswapService, liquidityService, hookService := server.InitializeServices(db)
	//tokenDeploymentHook, uniswapDeploymentHook := server.InitializeHooks(db, hookService)
	//
	//server.RegisterHooks(hookService, tokenDeploymentHook, uniswapDeploymentHook)
}
