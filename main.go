package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
)

const (
    maxFileSize = 50 * 1024 * 1024
)

func main() {
    var webhookURL string
    var count int

    fmt.Print("Discord WebhookのURLを入力してください：")
    fmt.Scanln(&webhookURL)

    if strings.TrimSpace(webhookURL) == "" {
        fmt.Println("Webhookが入力されていません。URLを一回だけ生成します。")
        postDiscord(webhookURL, false)
        return
    }

    fmt.Print("何個生成しますか？：")
    fmt.Scanln(&count)

    var wg sync.WaitGroup

    if count == 0 {
        for {
            wg.Add(1)
            go func() {
                defer wg.Done()
                postDiscord(webhookURL, true)
                time.Sleep(1 * time.Second)
            }()
        }
    } else {
        for i := 0; i < count; i++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                postDiscord(webhookURL, true)
            }()
        }
    }

    wg.Wait()
    fmt.Println("生成完了")

    if count > 0 {
        splitAndSendFile("Promo.txt", webhookURL)
    }
}

func postDiscord(webhookURL string, sendToWebhook bool) {
    client := &http.Client{}

    payload := map[string]string{
        "partnerUserId": "804e875e43164c862d446c880d316d675cdb44f8af843e41f9b759cef7ac9484",
    }
    jsonPayload, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", "https://api.discord.gx.games/v1/direct-fulfillment", bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("authority", "api.discord.gx.games")
    req.Header.Set("accept", "*/*")
    req.Header.Set("accept-language", "ja,en-US;q=0.9,en;q=0.8")
    req.Header.Set("origin", "https://www.opera.com")
    req.Header.Set("referer", "https://www.opera.com/")
    req.Header.Set("sec-ch-ua", `"Opera GX";v="105", "Chromium";v="119", "Not?A_Brand";v="24"`)
    req.Header.Set("sec-ch-ua-mobile", "?0")
    req.Header.Set("sec-ch-ua-platform", `"Windows"`)
    req.Header.Set("sec-fetch-dest", "empty")
    req.Header.Set("sec-fetch-mode", "cors")
    req.Header.Set("sec-fetch-site", "cross-site")
    req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36 OPR/105.0.0.0")

    resp, err := client.Do(req)
    if err != nil {
        fmt.Println("リクエストエラー:", err)
        return
    }
    defer resp.Body.Close()

    var data map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&data)

    if token, ok := data["token"].(string); ok {
        discordLink := fmt.Sprintf("https://discord.com/billing/partner-promotions/1180231712274387115/%s", token)
        appendToFile("Promo.txt", discordLink+"\n")

        if sendToWebhook {
            postWebhook(webhookURL, discordLink)
        }

        fmt.Println("promo Nitro:", discordLink)
    } else {
        fmt.Println("エラー")
    }
}

func appendToFile(filename, content string) {
    f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        fmt.Println("ファイルエラー:", err)
        return
    }
    defer f.Close()

    if _, err := f.WriteString(content); err != nil {
        fmt.Println("ファイルエラー:", err)
    }
}

func splitAndSendFile(filename, webhookURL string) {
    file, err := os.Open(filename)
    if err != nil {
        fmt.Println("ファイルエラー:", err)
        return
    }
    defer file.Close()

    fileInfo, err := file.Stat()
    if err != nil {
        fmt.Println("ファイルエラー:", err)
        return
    }

    fileSize := fileInfo.Size()
    if fileSize <= maxFileSize {
        sendFile(webhookURL, filename)
        return
    }

    buffer := make([]byte, maxFileSize)
    partNum := 0
    for {
        bytesRead, err := file.Read(buffer)
        if err != nil {
            if err != io.EOF {
                fmt.Println("ファイルエラー:", err)
                return
            }
            break
        }

        partFileName := fmt.Sprintf("%s.part%d", filename, partNum)
        partFile, err := os.Create(partFileName)
        if err != nil {
            fmt.Println("ファイルエラー:", err)
            return
        }

        _, err = partFile.Write(buffer[:bytesRead])
        partFile.Close()
        if err != nil {
            fmt.Println("ファイルエラー:", err)
            return
        }

        sendFile(webhookURL, partFileName)
        partNum++
    }
}

func sendFile(webhookURL, filename string) {
    file, err := os.Open(filename)
    if err != nil {
        fmt.Println("ファイルエラー:", err)
        return
    }
    defer file.Close()

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
    if err != nil {
        fmt.Println("ファイルエラー:", err)
        return
    }
    _, err = io.Copy(part, file)
    if err != nil {
        fmt.Println("ファイルエラー:", err)
        return
    }
    writer.Close()

    req, err := http.NewRequest("POST", webhookURL, body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    client := &http.Client{}
    _, err = client.Do(req)
    if err != nil {
        fmt.Println("Webhookエラー:", err)
        return
    }

    fmt.Println("送信完了:", filename)
}

func postWebhook(webhookURL, message string) {
    client := &http.Client{}
    payload := map[string]string{"content": message}
    jsonPayload, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonPayload))
    req.Header.Set("Content-Type", "application/json")

    _, err := client.Do(req)
    if err != nil {
        fmt.Println("Webhook:", err)
    }
}
