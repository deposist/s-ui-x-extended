package cmd

import (
	"fmt"

	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/util/common"
)

func resetAdmin() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	// Generate a random password instead of the well-known admin/admin so a reset
	// never leaves the panel on default credentials. Print it once for the
	// operator (it is stored only as a bcrypt hash).
	password := common.Random(16)
	userService := service.UserService{}
	if err := userService.UpdateFirstUser("admin", password); err != nil {
		fmt.Println("reset admin credentials failed:", err)
		return
	}
	fmt.Println("reset admin credentials success")
	fmt.Println("\tUsername:\tadmin")
	fmt.Printf("\tPassword:\t%s\n", password)
	fmt.Println("Save this password now; it cannot be recovered later.")
}

func updateAdmin(username string, password string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	if username != "" || password != "" {
		userService := service.UserService{}
		err := userService.UpdateFirstUser(username, password)
		if err != nil {
			fmt.Println("reset admin credentials failed:", err)
		} else {
			fmt.Println("reset admin credentials success")
		}
	}
}

func showAdmin() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}
	userService := service.UserService{}
	userModel, err := userService.GetFirstUser()
	if err != nil {
		fmt.Println("get current user info failed,error info:", err)
	}
	username := userModel.Username
	if username == "" || userModel.Password == "" {
		fmt.Println("current username or password is empty")
	}
	fmt.Println("First admin credentials:")
	fmt.Println("\tUsername:\t", username)
	fmt.Println("\tPassword is hashed; use 's-ui admin -reset' or 's-ui admin -username/-password' to set a new one")
}
