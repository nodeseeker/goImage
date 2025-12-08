package r2

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"hosting/internal/global"
)

// InitR2 初始化 Cloudflare R2 客户端
func InitR2() {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", global.AppConfig.R2.AccountID),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			global.AppConfig.R2.AccessKeyID,
			global.AppConfig.R2.AccessKeySecret,
			"",
		)),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Fatal(err)
	}

	global.R2Client = s3.NewFromConfig(cfg)
	log.Println("R2 client initialized successfully")
}

// UploadFile 上传文件到 R2
func UploadFile(ctx context.Context, key string, body io.Reader, contentType string) error {
	_, err := global.R2Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(global.AppConfig.R2.BucketName),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to R2: %w", err)
	}
	return nil
}

// DeleteFile 从 R2 删除文件
func DeleteFile(ctx context.Context, key string) error {
	_, err := global.R2Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(global.AppConfig.R2.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from R2: %w", err)
	}
	return nil
}

// GetPublicURL 获取文件的公开访问 URL
func GetPublicURL(key string) string {
	return fmt.Sprintf("%s/%s", global.AppConfig.R2.PublicURL, key)
}

// GetFile 从 R2 获取文件
func GetFile(ctx context.Context, key string) (*s3.GetObjectOutput, error) {
	result, err := global.R2Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(global.AppConfig.R2.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file from R2: %w", err)
	}
	return result, nil
}

// GetFileWithRange 从 R2 获取文件（支持 Range 请求）
func GetFileWithRange(ctx context.Context, key string, rangeHeader string) (*s3.GetObjectOutput, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(global.AppConfig.R2.BucketName),
		Key:    aws.String(key),
	}
	if rangeHeader != "" {
		input.Range = aws.String(rangeHeader)
	}
	result, err := global.R2Client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from R2: %w", err)
	}
	return result, nil
}
