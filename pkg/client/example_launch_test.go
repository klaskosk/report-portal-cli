package client_test

import (
	"context"
	"fmt"
	"log"

	"github.com/klaskosk/report-portal-cli/pkg/client"
)

func ExampleLaunchScope_GetByID() { //nolint:testableexamples // Output depends on live API.
	c, err := client.New("https://reportportal.example.com", "my-api-token")
	if err != nil {
		log.Fatal(err)
	}

	launch, err := c.Project("my_project").Launches().GetByID(context.Background(), 42)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(launch.Name)
}
