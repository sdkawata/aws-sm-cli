package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
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

const MAX_VERSION = 12

type versionEntry struct {
	stage string
	id    string
}

func removeStageForExcessVersions(client *secretsmanager.Client, id string) error {
	result, err := client.ListSecretVersionIds(context.Background(), &secretsmanager.ListSecretVersionIdsInput{
		SecretId:   aws.String(id),
		MaxResults: aws.Int32(100),
	})
	if err != nil {
		return err
	}
	// fmt.Printf("stageの数が%d個です\n", len(result.Versions))
	if len(result.Versions) < MAX_VERSION {
		return nil
	}
	targetVersions := make([]versionEntry, 0)
	for _, version := range result.Versions {
		nonTarget := false
		versionStage := ""
		for _, stage := range version.VersionStages {
			if stage == "AWSCURRENT" || stage == "AWSPREVIOUS" {
				nonTarget = true
				break
			}
			if strings.HasPrefix(stage, "VERSION_") {
				versionStage = stage
			}
		}
		if !nonTarget && versionStage != "" {
			targetVersions = append(targetVersions, versionEntry{
				stage: versionStage,
				id:    *version.VersionId,
			})
		}
	}
	sort.Slice(targetVersions, func(i, j int) bool {
		return targetVersions[i].stage < targetVersions[j].stage
	})
	for i := 0; i < len(targetVersions)-MAX_VERSION; i++ {
		fmt.Printf("stageが多すぎるので、stageを削除します%s\n", targetVersions[i].stage)
		_, err = client.UpdateSecretVersionStage(context.Background(), &secretsmanager.UpdateSecretVersionStageInput{
			SecretId:            aws.String(id),
			VersionStage:        aws.String(targetVersions[i].stage),
			MoveToVersionId:     nil,
			RemoveFromVersionId: aws.String(targetVersions[i].id),
		})
		if err != nil {
			return err
		}
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
	err = removeStageForExcessVersions(client, id)
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
