package objectstorage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const multipartPartSize int64 = 5 * 1024 * 1024

type Storage struct {
	client    *s3.Client
	presigner *s3.PresignClient
	uploader  *manager.Uploader
	bucket    string
}

func New(endpoint, region, accessKey, secretKey, bucket string, useSSL bool) (*Storage, error) {
	endpointURL, err := normalizeEndpoint(endpoint, useSSL)
	if err != nil {
		return nil, err
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load S3 configuration: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpointURL)
		options.UsePathStyle = true
	})
	return &Storage{
		client:    client,
		presigner: s3.NewPresignClient(client),
		uploader: manager.NewUploader(client, func(options *manager.Uploader) {
			options.PartSize = multipartPartSize
			options.Concurrency = 2
		}),
		bucket: bucket,
	}, nil
}

func normalizeEndpoint(endpoint string, useSSL bool) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("storage endpoint is required")
	}
	if !strings.Contains(endpoint, "://") {
		scheme := "http"
		if useSSL {
			scheme = "https"
		}
		endpoint = scheme + "://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("invalid storage endpoint")
	}
	return strings.TrimSuffix(endpoint, "/"), nil
}

func (s *Storage) PresignPut(ctx context.Context, key, contentType, sha256 string, expires time.Duration) (string, map[string]string, error) {
	signed, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"sha256": sha256,
		},
	}, func(options *s3.PresignOptions) {
		options.Expires = expires
	})
	if err != nil {
		return "", nil, err
	}
	return signed.URL, map[string]string{
		"Content-Type":      contentType,
		"X-Amz-Meta-Sha256": sha256,
	}, nil
}

func (s *Storage) InitiateMultipart(ctx context.Context, key, contentType, sha256 string, partCount int32, expires time.Duration) (string, map[int32]string, error) {
	created, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"sha256": sha256,
		},
	})
	if err != nil {
		return "", nil, err
	}
	uploadID := aws.ToString(created.UploadId)
	parts := make(map[int32]string, partCount)
	for partNumber := int32(1); partNumber <= partCount; partNumber++ {
		signed, signErr := s.presigner.PresignUploadPart(ctx, &s3.UploadPartInput{
			Bucket:     aws.String(s.bucket),
			Key:        aws.String(key),
			PartNumber: aws.Int32(partNumber),
			UploadId:   aws.String(uploadID),
		}, func(options *s3.PresignOptions) {
			options.Expires = expires
		})
		if signErr != nil {
			_ = s.AbortMultipart(ctx, key, uploadID)
			return "", nil, signErr
		}
		parts[partNumber] = signed.URL
	}
	return uploadID, parts, nil
}

func (s *Storage) CompleteMultipart(ctx context.Context, key, uploadID string, etags map[int32]string) error {
	partNumbers := make([]int, 0, len(etags))
	for partNumber := range etags {
		partNumbers = append(partNumbers, int(partNumber))
	}
	slices.Sort(partNumbers)
	parts := make([]types.CompletedPart, 0, len(partNumbers))
	for _, partNumber := range partNumbers {
		number := int32(partNumber)
		parts = append(parts, types.CompletedPart{
			ETag:       aws.String(etags[number]),
			PartNumber: aws.Int32(number),
		})
	}
	_, err := s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	return err
}

func (s *Storage) AbortMultipart(ctx context.Context, key, uploadID string) error {
	_, err := s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	return err
}

func (s *Storage) PresignGet(ctx context.Context, key, filename, disposition, contentType string, expires time.Duration) (string, error) {
	signed, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(s.bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(mime.FormatMediaType(disposition, map[string]string{"filename": filename})),
		ResponseContentType:        aws.String(contentType),
	}, func(options *s3.PresignOptions) {
		options.Expires = expires
	})
	if err != nil {
		return "", err
	}
	return signed.URL, nil
}

func (s *Storage) Stat(ctx context.Context, key string) (int64, string, map[string]string, error) {
	info, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return 0, "", nil, err
	}
	return aws.ToInt64(info.ContentLength), aws.ToString(info.ContentType), info.Metadata, nil
}

func (s *Storage) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err == nil {
		return nil
	}
	_, createErr := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)})
	return createErr
}

func (s *Storage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          reader,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	return err
}

func (s *Storage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return object.Body, nil
}

func (s *Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
