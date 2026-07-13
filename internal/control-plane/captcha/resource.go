// 加载 go-captcha 滑块验证码所需的背景图和拼图块资源
// 背景图从自定义图片目录加载，拼图块从 tile 目录加载预置图片
package captcha

import (
	"image"
	"os"
	"path/filepath"
	"strings"

	"github.com/wenlng/go-captcha/v2/slide"
)

// ImageWidth 滑块验证码主图宽度
const ImageWidth = 300

// ImageHeight 滑块验证码主图高度
const ImageHeight = 200

// LoadBackgrounds 从目录加载自定义背景图（支持 PNG/JPEG/GIF）
// 图片尺寸应 >= 300x200，go-captcha 会自动随机裁剪匹配区域
func LoadBackgrounds(dir string) ([]image.Image, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var imgs []image.Image
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" {
			continue
		}
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		img, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			continue
		}
		imgs = append(imgs, img)
	}
	return imgs, nil
}

// LoadGraphImages 从 tileDir 目录加载预置的拼图块图片（overlay、shadow、mask）
// 目录中需包含 tile.png（覆盖图）、tile-shadow.png（阴影）、tile-mask.png（遮罩）
func LoadGraphImages(tileDir string) ([]*slide.GraphImage, error) {
	overlayImg, err := decodeImage(filepath.Join(tileDir, "tile.png"))
	if err != nil {
		return nil, err
	}
	shadowImg, err := decodeImage(filepath.Join(tileDir, "tile-shadow.png"))
	if err != nil {
		return nil, err
	}
	maskImg, err := decodeImage(filepath.Join(tileDir, "tile-mask.png"))
	if err != nil {
		return nil, err
	}

	return []*slide.GraphImage{
		{
			OverlayImage: overlayImg,
			ShadowImage:  shadowImg,
			MaskImage:    maskImg,
		},
	}, nil
}

// decodeImage 从文件路径解码一张图片
func decodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}
