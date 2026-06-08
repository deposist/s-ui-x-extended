package cmd

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/deposist/s-ui-x/cmd/migration"
	"github.com/deposist/s-ui-x/config"
)

func ParseCmd() {
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "show version")

	adminCmd := flag.NewFlagSet("admin", flag.ExitOnError)
	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)
	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)

	var username string
	var password string
	var port int
	var path string
	var subPort int
	var subPath string
	var reset bool
	var show bool
	var clearDomain bool
	var repairFKOrphans bool
	settingCmd.BoolVar(&reset, "reset", false, "reset all settings")
	settingCmd.BoolVar(&show, "show", false, "show current settings")
	settingCmd.BoolVar(&clearDomain, "clearDomain", false, "clear panel domain, listen address and web URI")
	settingCmd.IntVar(&port, "port", 0, "set panel port")
	settingCmd.StringVar(&path, "path", "", "set panel path")
	settingCmd.IntVar(&subPort, "subPort", 0, "set sub port")
	settingCmd.StringVar(&subPath, "subPath", "", "set sub path")
	migrateCmd.BoolVar(&repairFKOrphans, "repair-fk-orphans", false, "delete safe foreign-key orphans during migration")

	adminCmd.BoolVar(&show, "show", false, "show first admin credentials")
	adminCmd.BoolVar(&reset, "reset", false, "reset first admin credentials")
	adminCmd.StringVar(&username, "username", "", "set login username")
	adminCmd.StringVar(&password, "password", "", "set login password")

	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("    admin          set/reset/show first admin credentials")
		fmt.Println("    decrypt-backup decrypt Telegram backup envelope")
		fmt.Println("    import-xui     import configuration from a 3x-ui database")
		fmt.Println("    uri            Show panel URI")
		fmt.Println("    migrate        migrate form older version")
		fmt.Println("    setting        set/reset/clear/show settings")
		fmt.Println()
		adminCmd.Usage()
		fmt.Println()
		settingCmd.Usage()
		fmt.Println()
		migrateCmd.Usage()
	}

	flag.Parse()
	if showVersion {
		fmt.Println("S-UI Panel\t", config.GetVersion())
		info, ok := debug.ReadBuildInfo()
		if ok {
			for _, dep := range info.Deps {
				if dep.Path == "github.com/sagernet/sing-box" {
					fmt.Println("Sing-Box\t", dep.Version)
					break
				}
			}
		}
		return
	}

	switch os.Args[1] {
	case "admin":
		err := adminCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case show:
			showAdmin()
		case reset:
			resetAdmin()
		default:
			updateAdmin(username, password)
			showAdmin()
		}

	case "uri":
		getPanelURI()

	case "migrate":
		if err := migrateCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println(err)
			return
		}
		if err := migration.MigrateDbWithOptions(migration.Options{RepairForeignKeyOrphans: repairFKOrphans}); err != nil {
			fmt.Println("migrate failed:", err)
			os.Exit(1)
		}

	case "import-xui":
		os.Exit(runImportXui(os.Args[2:], os.Stdout))

	case "decrypt-backup":
		os.Exit(runDecryptBackup(os.Args[2:], os.Stdin, os.Stdout, os.Stderr, os.Getenv))

	case "setting":
		err := settingCmd.Parse(os.Args[2:])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case show:
			showSetting()
		case reset:
			resetSetting()
		case clearDomain:
			clearWebDomain()
		default:
			updateSetting(port, path, subPort, subPath)
			showSetting()
		}
	default:
		fmt.Println("Invalid subcommands")
		flag.Usage()
	}
}
