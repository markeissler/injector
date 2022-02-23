package gcp

import (
	"encoding/base64"
	"fmt"
	"io"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"

	"github.com/markeissler/injector/pkg/stringutil"
)

// FetchSecretDocument retrieves the secret manager document specified in cli arguments and writes the contents to the
// specified io.Writer. The `latest` version will be retrieved if no version has been specified.
func FetchSecretDocument(ctx *cli.Context, writer io.Writer) error {
	var client *secretmanager.Client
	var err error

	// Set the secret manager Client option for reading credentials from a file.
	clientOptions := make([]option.ClientOption, 0)
	if !stringutil.IsBlank(ctx.String("key-file")) {
		clientOptions = append(clientOptions, option.WithCredentialsFile(ctx.String("key-file")))
	} else if !stringutil.IsBlank(ctx.String("key-value")) {
		var jsonBytes []byte
		jsonBytes, err = base64.StdEncoding.DecodeString(ctx.String("key-value"))
		if err != nil {
			return fmt.Errorf("failed to decode secretmanager service account key value: %v", err)
		}
		clientOptions = append(clientOptions, option.WithCredentialsJSON(jsonBytes))
	}

	// Create the secret manager Client.
	if client, err = secretmanager.NewClient(ctx.Context, clientOptions...); err != nil {
		return fmt.Errorf("failed to create secretmanager client: %v", err)
	}
	defer func() {
		if err = client.Close(); err != nil {
			log.Println("error encountered while cleaning up secretManager.Client")
		}
	}()

	// Build the request.
	secretVersion := "latest"
	if !stringutil.IsBlank(ctx.String("secret-version")) {
		secretVersion = ctx.String("secret-version")
	}
	secretName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", ctx.String("project"), ctx.String("secret-name"), secretVersion)
	request := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	// Call the API.
	var result *secretmanagerpb.AccessSecretVersionResponse
	if result, err = client.AccessSecretVersion(ctx.Context, request); err != nil {
		return fmt.Errorf("failed to access secret version: %v", err)
	}

	// Write the contents to the io.Writer.
	if _, err = fmt.Fprintf(writer, "%s\n", string(result.Payload.Data)); err != nil {
		return err
	}

	return nil
}
