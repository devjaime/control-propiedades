package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type storageConfig struct {
	endpoint  string
	region    string
	accessKey string
	secretKey string
	bucket    string
}

type objectInfo struct {
	key  string
	size int64
}

func main() {
	ctx := context.Background()
	sourceConfig := loadStorageConfig("SOURCE")
	targetConfig := loadStorageConfig("TARGET")
	source := newClient(ctx, sourceConfig)
	target := newClient(ctx, targetConfig)
	uploader := manager.NewUploader(target, func(options *manager.Uploader) {
		options.PartSize = 5 * 1024 * 1024
		options.Concurrency = 2
	})
	if err := ensureBucket(ctx, target, targetConfig.bucket); err != nil {
		log.Fatalf("preparar bucket de destino: %v", err)
	}

	objects, err := listObjects(ctx, source, sourceConfig.bucket)
	if err != nil {
		log.Fatalf("listar almacenamiento de origen: %v", err)
	}

	var copiedBytes int64
	for index, object := range objects {
		if err := copyObject(ctx, source, uploader, sourceConfig.bucket, targetConfig.bucket, object); err != nil {
			log.Fatalf("copiar objeto %q: %v", object.key, err)
		}
		copiedBytes += object.size
		fmt.Printf("copiados=%d/%d\r", index+1, len(objects))
	}
	if len(objects) > 0 {
		fmt.Println()
	}

	targetObjects, err := listObjects(ctx, target, targetConfig.bucket)
	if err != nil {
		log.Fatalf("verificar almacenamiento de destino: %v", err)
	}
	if err := compareObjects(objects, targetObjects); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("migracion_verificada objetos=%d bytes=%d\n", len(objects), copiedBytes)
}

func ensureBucket(ctx context.Context, client *s3.Client, bucket string) error {
	if _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err == nil {
		return nil
	}
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	return err
}

func loadStorageConfig(prefix string) storageConfig {
	value := func(name string) string {
		result := strings.TrimSpace(os.Getenv(prefix + "_STORAGE_" + name))
		if result == "" {
			log.Fatalf("falta %s_STORAGE_%s", prefix, name)
		}
		return result
	}
	return storageConfig{
		endpoint:  value("ENDPOINT"),
		region:    value("REGION"),
		accessKey: value("ACCESS_KEY"),
		secretKey: value("SECRET_KEY"),
		bucket:    value("BUCKET"),
	}
}

func newClient(ctx context.Context, cfg storageConfig) *s3.Client {
	endpoint := cfg.endpoint
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	loaded, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.accessKey, cfg.secretKey, "")),
	)
	if err != nil {
		log.Fatalf("configurar cliente S3: %v", err)
	}
	return s3.NewFromConfig(loaded, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(strings.TrimSuffix(endpoint, "/"))
		options.UsePathStyle = true
	})
}

func listObjects(ctx context.Context, client *s3.Client, bucket string) ([]objectInfo, error) {
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	var objects []objectInfo
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, object := range page.Contents {
			objects = append(objects, objectInfo{key: aws.ToString(object.Key), size: aws.ToInt64(object.Size)})
		}
	}
	sort.Slice(objects, func(i, j int) bool { return objects[i].key < objects[j].key })
	return objects, nil
}

func copyObject(ctx context.Context, source *s3.Client, target *manager.Uploader, sourceBucket, targetBucket string, object objectInfo) error {
	download, err := source.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(sourceBucket),
		Key:    aws.String(object.key),
	})
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(download.Body)
	closeErr := download.Body.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if int64(len(data)) != object.size {
		return fmt.Errorf("tamaño leído %d, esperado %d", len(data), object.size)
	}

	_, err = target.Upload(ctx, &s3.PutObjectInput{
		Bucket:             aws.String(targetBucket),
		Key:                aws.String(object.key),
		Body:               bytes.NewReader(data),
		ContentLength:      aws.Int64(int64(len(data))),
		ContentType:        download.ContentType,
		ContentDisposition: download.ContentDisposition,
		Metadata:           download.Metadata,
	})
	return err
}

func compareObjects(source, target []objectInfo) error {
	if len(source) != len(target) {
		return fmt.Errorf("verificación fallida: origen=%d destino=%d", len(source), len(target))
	}
	for index := range source {
		if source[index] != target[index] {
			return fmt.Errorf("verificación fallida en %q", source[index].key)
		}
	}
	return nil
}
