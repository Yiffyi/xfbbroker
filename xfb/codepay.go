package xfb

import (
	"fmt"
	"image/png"
	"net/url"
	"os"
	"time"

	"github.com/skip2/go-qrcode"
)

const (
	qrCodeEndpointUrl   = "/card/getQRCode"
	qrResultEndpointUrl = "/card/getQRCodeResult"
)

type QrPayCode struct {
	QRCode    string
	SessionID string
	Creation  int64
}

func GenerateQrPayCode(sessionId string) (*QrPayCode, error) {
	var result XfbResponse

	_, err := Post(XfbWebApp+qrCodeEndpointUrl, sessionId, nil, &result)
	if err != nil {
		return nil, err
	}

	if result.StatusCode != 0 {
		return nil, fmt.Errorf("API error: statusCode=%d, statusCode=%d",
			result.StatusCode, result.StatusCode)
	}

	return &QrPayCode{
		QRCode:    result.Data.(string),
		SessionID: sessionId,
		Creation:  time.Now().Unix(),
	}, nil
}

func (q *QrPayCode) SaveQrImage(filename string) error {
	qr, err := qrcode.New(q.QRCode, qrcode.Medium)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, qr.Image(256))
}

func (q *QrPayCode) GetQrPngBuf(size int) ([]byte, error) {
	qr, err := qrcode.New(q.QRCode, qrcode.Medium)
	if err != nil {
		return nil, err
	}

	return qr.PNG(size)
}

func (q *QrPayCode) GetResult() (map[string]any, error) {
	form := url.Values{}
	form.Add("qrCode", q.QRCode)

	var result XfbResponse

	_, err := PostForm(XfbWebApp+qrResultEndpointUrl, q.SessionID, form, &result)
	if err != nil {
		return nil, err
	}

	return result.Data.(map[string]any), nil
}

// func main() {
// 	if len(os.Args) < 2 {
// 		fmt.Println("Usage: program <session-id>")
// 		os.Exit(1)
// 	}

// 	qr, err := GenerateQrPayCode(os.Args[1])
// 	if err != nil {
// 		fmt.Printf("Error generating QR code: %v\n", err)
// 		os.Exit(1)
// 	}

// 	if err := qr.SaveQrImage("qrcode.png"); err != nil {
// 		fmt.Printf("Error saving QR code: %v\n", err)
// 		os.Exit(1)
// 	}

// 	timeout := time.After(totalCheckTime)
// 	ticker := time.NewTicker(checkInterval)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			result, err := qr.GetResult()
// 			if err != nil {
// 				fmt.Printf("Error checking result: %v\n", err)
// 				continue
// 			}

// 			fmt.Println("Check result:", result)
// 			if success, ok := result["success"].(bool); ok && success {
// 				fmt.Println("Payment successful!")
// 				return
// 			}

// 		case <-timeout:
// 			fmt.Println("Timeout waiting for payment confirmation")
// 			return
// 		}
// 	}
// }
