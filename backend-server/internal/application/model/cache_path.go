package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// CachePathInfo URLとContentTypeからファイルパス情報を生成する（domain層のロジック）
type CachePathInfo struct {
	Host     string // サニタイズ済みのホスト名
	Path     string // サニタイズ済みのパス
	SubDir   string // ContentTypeに応じたサブディレクトリ（css, js, images, fontsなど）
	FileName string // ファイル名
}

// GenerateCachePathInfo URLとContentTypeからキャッシュパス情報を生成する（domain層のロジック）
func GenerateCachePathInfo(resourceURL string, contentType string, cacheKey string) (*CachePathInfo, error) {
	if resourceURL == "" {
		return nil, fmt.Errorf("resource URL is empty")
	}

	// URLをパース
	parsedURL, err := url.Parse(resourceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// ホスト名を取得
	host := parsedURL.Host
	if host == "" {
		return nil, fmt.Errorf("URL host is empty")
	}

	// ホスト名を安全なファイル名に変換
	host = sanitizeForPath(host)

	// パスを取得（クエリパラメータとフラグメントは除外）
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	// パスを安全なディレクトリパスに変換
	path = sanitizeForPath(path)

	// ContentTypeから拡張子とサブディレクトリを決定
	ext := getExtensionFromContentType(contentType)
	subDir := getSubDirectoryFromContentType(contentType)

	// ファイル名を決定
	var fileName string
	if strings.HasSuffix(parsedURL.Path, "/") || parsedURL.Path == "" || parsedURL.Path == "/" {
		// ディレクトリの場合はindex.html
		if ext == ".html" {
			fileName = "index.html"
		} else {
			// その他の場合はハッシュを使用
			hash := generateHash(cacheKey)
			fileName = hash + ext
		}
	} else {
		// パスからファイル名を抽出
		baseName := filepath.Base(parsedURL.Path)
		// 既存の拡張子を除去してから新しい拡張子を追加
		if idx := strings.LastIndex(baseName, "."); idx != -1 {
			baseName = baseName[:idx]
		}
		fileName = baseName + ext
	}

	return &CachePathInfo{
		Host:     host,
		Path:     path,
		SubDir:   subDir,
		FileName: fileName,
	}, nil
}

// sanitizeForPath パスに使用できない文字を置換して安全なパスに変換（domain層のロジック）
// ただし、"/"は保持して階層構造を維持する
func sanitizeForPath(path string) string {
	// パストラバーサル攻撃を防ぐため、".." を削除
	path = strings.ReplaceAll(path, "..", "")

	// パスを"/"で分割して各部分を処理
	parts := strings.Split(path, "/")
	var sanitizedParts []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		// 特殊文字をアンダースコアに置換（"/"は除く）
		replacer := strings.NewReplacer(
			"\\", "_",
			":", "_",
			"*", "_",
			"?", "_",
			"\"", "_",
			"<", "_",
			">", "_",
			"|", "_",
		)
		sanitizedPart := replacer.Replace(part)
		// 先頭と末尾のアンダースコアを削除
		sanitizedPart = strings.Trim(sanitizedPart, "_")
		if sanitizedPart != "" {
			sanitizedParts = append(sanitizedParts, sanitizedPart)
		}
	}

	// パスを再構築
	if len(sanitizedParts) == 0 {
		return "root"
	}
	return strings.Join(sanitizedParts, "/")
}

// getSubDirectoryFromContentType ContentTypeからサブディレクトリを決定（domain層のロジック）
func getSubDirectoryFromContentType(contentType string) string {
	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(mainType)

	// Content-Typeからサブディレクトリをマッピング
	contentTypeToSubDir := map[string]string{
		"text/css":                    "css",
		"application/javascript":      "js",
		"application/x-javascript":    "js",
		"text/javascript":             "js",
		"image/jpeg":                  "images",
		"image/jpg":                   "images",
		"image/png":                   "images",
		"image/gif":                   "images",
		"image/svg+xml":               "images",
		"image/webp":                  "images",
		"image/x-icon":                "images",
		"image/vnd.microsoft.icon":    "images",
		"font/woff":                   "fonts",
		"font/woff2":                  "fonts",
		"application/font-woff":       "fonts",
		"application/font-woff2":      "fonts",
		"font/ttf":                    "fonts",
		"application/x-font-ttf":      "fonts",
		"font/otf":                    "fonts",
		"application/x-font-opentype": "fonts",
	}

	// 完全一致をチェック
	if subDir, ok := contentTypeToSubDir[contentType]; ok {
		return subDir
	}

	// メインタイプでチェック
	if subDir, ok := contentTypeToSubDir[mainType]; ok {
		return subDir
	}

	// プレフィックスでチェック
	if strings.HasPrefix(mainType, "image/") {
		return "images"
	}
	if strings.HasPrefix(mainType, "font/") || strings.HasPrefix(mainType, "application/font") {
		return "fonts"
	}

	// デフォルトは空（HTMLなどはルートディレクトリ）
	return ""
}

// getExtensionFromContentType ContentTypeから適切な拡張子を取得（domain層のロジック）
func getExtensionFromContentType(contentType string) string {
	// Content-Typeヘッダーからメインタイプを抽出（例: "text/html; charset=utf-8" -> "text/html"）
	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(mainType)

	// Content-Typeから拡張子をマッピング
	contentTypeToExt := map[string]string{
		// HTML
		"text/html":                ".html",
		"text/html; charset=utf-8": ".html",
		"text/html; charset=UTF-8": ".html",

		// CSS
		"text/css":                ".css",
		"text/css; charset=utf-8": ".css",

		// JavaScript
		"application/javascript":         ".js",
		"application/x-javascript":       ".js",
		"text/javascript":                ".js",
		"text/javascript; charset=utf-8": ".js",

		// JSON
		"application/json":                ".json",
		"application/json; charset=utf-8": ".json",

		// XML
		"application/xml":         ".xml",
		"text/xml":                ".xml",
		"text/xml; charset=utf-8": ".xml",

		// 画像
		"image/jpeg":               ".jpg",
		"image/jpg":                ".jpg",
		"image/png":                ".png",
		"image/gif":                ".gif",
		"image/svg+xml":            ".svg",
		"image/webp":               ".webp",
		"image/x-icon":             ".ico",
		"image/vnd.microsoft.icon": ".ico",

		// フォント
		"font/woff":                   ".woff",
		"font/woff2":                  ".woff2",
		"application/font-woff":       ".woff",
		"application/font-woff2":      ".woff2",
		"font/ttf":                    ".ttf",
		"application/x-font-ttf":      ".ttf",
		"font/otf":                    ".otf",
		"application/x-font-opentype": ".otf",

		// その他
		"application/pdf":           ".pdf",
		"text/plain":                ".txt",
		"text/plain; charset=utf-8": ".txt",
		"application/octet-stream":  ".bin",
	}

	// 完全一致をチェック
	if ext, ok := contentTypeToExt[contentType]; ok {
		return ext
	}

	// メインタイプでチェック
	if ext, ok := contentTypeToExt[mainType]; ok {
		return ext
	}

	// メインタイプのプレフィックスでチェック（例: "image/*"）
	if strings.HasPrefix(mainType, "image/") {
		// 画像タイプの場合は拡張子を推測
		parts := strings.Split(mainType, "/")
		if len(parts) == 2 {
			subtype := parts[1]
			switch subtype {
			case "jpeg", "jpg":
				return ".jpg"
			case "png":
				return ".png"
			case "gif":
				return ".gif"
			case "svg+xml":
				return ".svg"
			case "webp":
				return ".webp"
			case "x-icon", "vnd.microsoft.icon":
				return ".ico"
			}
		}
	}

	// デフォルトは.html（HTMLページとして扱う）
	return ".html"
}

// generateHash キャッシュキーからハッシュを生成（domain層のロジック）
func generateHash(cacheKey string) string {
	hash := sha256.Sum256([]byte(cacheKey))
	return hex.EncodeToString(hash[:])
}
