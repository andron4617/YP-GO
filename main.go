package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	statsURL         = "http://srv.msk01.gigacorp.local/_stats"
	memHighPercent   = 80
	diskHighPercent  = 90
	netHighPercent   = 90
)

func main() {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	errorCount := 0

	for {
		ok := pollOnce(client)
		if !ok {
			errorCount++
			if errorCount >= 3 {
				fmt.Fprintln(os.Stdout, "Unable to fetch server statistic")
			}
		} else {
			// если данные удалось получить и распарсить — сбрасываем счётчик ошибок
			errorCount = 0
		}

		time.Sleep(1 * time.Second)
	}
}

// pollOnce делает один запрос к серверу и выводит сообщения по порогам.
// Возвращает true, если данные корректны, false — если ошибка (HTTP, парсинг и т.п.).
func pollOnce(client *http.Client) bool {
	resp, err := client.Get(statsURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	text := strings.TrimSpace(string(body))
	fields := strings.Split(text, ",")
	if len(fields) != 7 {
		return false
	}

	loadAvg, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return false
	}

	memTotal, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil || memTotal == 0 {
		return false
	}
	memUsed, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return false
	}

	diskTotal, err := strconv.ParseUint(fields[3], 10, 64)
	if err != nil || diskTotal == 0 {
		return false
	}
	diskUsed, err := strconv.ParseUint(fields[4], 10, 64)
	if err != nil {
		return false
	}

	netTotal, err := strconv.ParseUint(fields[5], 10, 64)
	if err != nil || netTotal == 0 {
		return false
	}
	netUsed, err := strconv.ParseUint(fields[6], 10, 64)
	if err != nil {
		return false
	}

	checkThresholds(loadAvg, memTotal, memUsed, diskTotal, diskUsed, netTotal, netUsed)
	return true
}

// checkThresholds проверяет все пороги и печатает сообщения в stdout.
func checkThresholds(loadAvg float64, memTotal, memUsed, diskTotal, diskUsed, netTotal, netUsed uint64) {
	// 1. Load Average
	if loadAvg > 30 {
		fmt.Fprintf(os.Stdout, "Load Average is too high: %v\n", loadAvg)
	}

	// 3. Память: > 80% от общего объёма
	memPercent := memUsed * 100 / memTotal
	if memPercent > memHighPercent {
		fmt.Fprintf(os.Stdout, "Memory usage too high: %d%%\n", memPercent)
	}

	// 5. Диск: > 90% использования, считаем свободное место в мегабайтах
	diskPercent := diskUsed * 100 / diskTotal
	if diskPercent > diskHighPercent {
		freeBytes := diskTotal - diskUsed
		freeMB := freeBytes / (1024 * 1024)
		fmt.Fprintf(os.Stdout, "Free disk space is too low: %d Mb left\n", freeMB)
	}

	// 7. Сеть: > 90% использования
netPercent := netUsed * 100 / netTotal
if netPercent > netHighPercent {
    freeBytes := netTotal - netUsed
    freeMbit := freeBytes / 1_000_000
    fmt.Fprintf(os.Stdout, "Network bandwidth usage high: %d Mbit/s available\n", freeMbit)
}
}
