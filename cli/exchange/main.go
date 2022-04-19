package exchange

import "fmt"

func Main() {
	setup()

	err := rootCmd.Execute()
	if err != nil {
		fmt.Println("Error:", err)
	}
}
