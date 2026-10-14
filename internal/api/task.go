package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/gzjacktang/single-ui/internal/task"
)

func TaskStatus(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	t := task.Get(c.Param("id"))
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在或已过期，请刷新列表查看结果", "expired": true})
		return
	}
	after, _ := strconv.Atoi(c.Query("after"))
	c.JSON(http.StatusOK, t.Snapshot(after))
}

func TaskLog(c *gin.Context) {
	id := c.Param("id")
	t := task.Get(id)
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for line := range t.Chan() {
		parts := strings.SplitN(line, " ", 4)
		var ll logLine
		if len(parts) >= 4 {
			ll = logLine{Time: parts[0] + " " + parts[1], Level: parts[2], Message: parts[3]}
		} else {
			ll = logLine{Level: "INFO", Message: line}
		}
		data, _ := json.Marshal(ll)
		if conn.WriteMessage(websocket.TextMessage, data) != nil {
			return
		}
	}

	data, _ := json.Marshal(logLine{Level: "DONE", Message: "任务完成"})
	conn.WriteMessage(websocket.TextMessage, data)
}
