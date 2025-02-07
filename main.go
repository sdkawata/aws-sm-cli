package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	jd "github.com/josephburnett/jd/lib"
	"github.com/urfave/cli"
)

func waitYes() error {
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	if text != "yes\n" {
		return cli.NewExitError(fmt.Errorf("中断します"), 1)
	}
	return nil
}

func dumpAction(c *cli.Context) error {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	client := secretsmanager.NewFromConfig(cfg)
	id := c.String("id")
	if id == "" {
		return cli.NewExitError("idは必須です", 1)
	}
	result, err := client.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(id),
	})
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	buf := bytes.NewBuffer([]byte{})
	json.Indent(buf, []byte(*result.SecretString), "", "  ")
	fileName := c.String("file")
	if fileName != "" {
		fileName = ".env"
	}
	err = os.WriteFile(fileName, buf.Bytes(), 0644)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	return nil
}

func changeAction(c *cli.Context) error {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	client := secretsmanager.NewFromConfig(cfg)
	id := c.String("id")
	if id == "" {
		return cli.NewExitError("idは必須です", 1)
	}
	result, err := client.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(id),
	})
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	fileName := c.String("file")
	if fileName == "" {
		fileName = ".env"
	}
	file, err := os.ReadFile(fileName)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	original, err := jd.ReadJsonString(*result.SecretString)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	new, err := jd.ReadJsonString(string(file))
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	diff := original.Diff(new)
	if diff.Render() == "" {
		fmt.Println("変更はありません")
		return nil
	}
	fmt.Print(original.Diff(new).Render())
	fmt.Println("変更を適応するならyesと入力してください")
	err = waitYes()
	if err != nil {
		return err
	}
	_, err = client.PutSecretValue(context.Background(), &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(id),
		SecretString: aws.String(string(file)),
		VersionStages: []string{
			"AWSCURRENT",
			"VERSION_" + time.Now().Format("20060102150405"),
		},
	})
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	fmt.Println("更新が完了しました")
	return nil
}

func revertAction(c *cli.Context) error {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	client := secretsmanager.NewFromConfig(cfg)
	id := c.String("id")
	if id == "" {
		return cli.NewExitError("idは必須です", 1)
	}
	result, err := client.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(id),
	})
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	prevResult, err := client.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(id),
		VersionStage: aws.String("AWSPREVIOUS"),
	})
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	original, err := jd.ReadJsonString(*result.SecretString)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	new, err := jd.ReadJsonString(*prevResult.SecretString)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	fmt.Print(original.Diff(new).Render())
	fmt.Println("変更を適応するならyesと入力してください")
	err = waitYes()
	if err != nil {
		return err
	}
	_, err = client.UpdateSecretVersionStage(context.Background(), &secretsmanager.UpdateSecretVersionStageInput{
		SecretId:            aws.String(id),
		VersionStage:        aws.String("AWSCURRENT"),
		MoveToVersionId:     aws.String(*prevResult.VersionId),
		RemoveFromVersionId: aws.String(*result.VersionId),
	})
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	fmt.Println("更新が完了しました")
	return nil
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name: "dump",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "id",
					Usage: "secret id",
				},
				cli.StringFlag{
					Name: "file,f",
				},
			},
			Action: dumpAction,
		},
		{
			Name: "change",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "id",
					Usage: "secret id",
				},
				cli.StringFlag{
					Name: "file,f",
				},
			},
			Action: changeAction,
		},
		{
			Name: "revert",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "id",
					Usage: "secret id",
				},
			},
			Action: revertAction,
		},
	}
	app.Run(os.Args)
}
