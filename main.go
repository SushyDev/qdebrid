package main

import (
	"qdebrid/logger"
	"qdebrid/qbittorrent"
)

var sugar = logger.Sugar()

func main() {
	sugar.Info("Program started")

	qbittorrent.Listen()
}
