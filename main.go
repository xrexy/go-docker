package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/gofiber/fiber/v2"
)

func main() {

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	defer cli.Close()

	app := fiber.New()
	dkGroup := app.Group("docker")
	dkGroup.Get("/containers", func(c *fiber.Ctx) error {
		containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
		if err != nil {
			return c.SendStatus(400)
		}

		for _, container := range containers {
			fmt.Printf("%s %s\n", container.ID[:10], container.Image)
		}
		return c.JSON(fiber.Map{
			"containers": fmt.Sprintf("%s %s\n", containers[0].ID[:10], containers[0].Image),
		})
	})

	dkGroup.Post("/container", func(c *fiber.Ctx) error {
		reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
		if err != nil {
			panic(err)
		}

		defer reader.Close()
		io.Copy(os.Stdout, reader)

		resp, err := cli.ContainerCreate(ctx, &container.Config{
			Image: "alpine",
			Cmd:   []string{"echo", "hello world"},
			Tty:   false,
		}, nil, nil, nil, fmt.Sprintf("test-%d", time.Now().Unix()))
		if err != nil {
			panic(err)
		}

		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			panic(err)
		}

		statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				panic(err)
			}
		case <-statusCh:
		}

		out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			panic(err)
		}

		stdcopy.StdCopy(os.Stdout, os.Stderr, out)

		return c.SendStatus(200)
	})

	if err := app.Listen(":3000"); err != nil {
		panic(err)
	}
}
