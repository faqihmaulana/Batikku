package web

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "mime/multipart"
    "net/http"
    "os"
	"io"
    "path/filepath"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
)

func main() {
    gin.SetMode(gin.ReleaseMode) // mode rilis

    wg := sync.WaitGroup{}

    wg.Add(1)
    go func() {
        defer wg.Done()

        router := gin.New()
        router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
            return fmt.Sprintf("[%s] \"%s %s %s\"\n",
                param.TimeStamp.Format(time.RFC822),
                param.Method,
                param.Path,
                param.ErrorMessage,
            )
        }))
        router.Use(gin.Recovery())

        router.StaticFS("/static", http.Dir("frontend/public"))

        router.GET("/", func(c *gin.Context) {
            c.File("views/prediksi-batik.html")
        })

        router.POST("/api/v1/predict-batik", func(c *gin.Context) {
            file, err := c.FormFile("file")
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada file yang diterima"})
                return
            }

            // Simpan file secara lokal
            localPath := "./uploads/" + file.Filename
            if err := c.SaveUploadedFile(file, localPath); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat menyimpan file"})
                return
            }

            // Panggil API prediksi
            prediction, err := predictBatik(localPath)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat mendapatkan prediksi"})
                return
            }

            // Hapus file lokal
            os.Remove(localPath)

            c.JSON(http.StatusOK, gin.H{"prediction": prediction})
        })

        fmt.Println("Server berjalan di port 8080")
        err := router.Run(":8080")
        if err != nil {
            panic(err)
        }
    }()

    wg.Wait()
}

func predictBatik(filePath string) (string, error) {
    url := "http://127.0.0.1:5000/predict"

    // Buka file
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    // Buat form yang meniru unggahan file multipart
    var b bytes.Buffer
    w := multipart.NewWriter(&b)
    fw, err := w.CreateFormFile("image", filepath.Base(file.Name()))
    if err != nil {
        return "", err
    }
    if _, err = ioutil.ReadAll(file); err != nil {
        return "", err
    }
    if _, err = io.Copy(fw, file); err != nil {
        return "", err
    }
    w.Close()

    // Buat permintaan
    req, err := http.NewRequest("POST", url, &b)
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", w.FormDataContentType())

    client := &http.Client{}
    res, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer res.Body.Close()

    if res.StatusCode != http.StatusOK {
        return "", fmt.Errorf("menerima kode respons non-200")
    }

    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        return "", err
    }

    return string(body), nil
}
