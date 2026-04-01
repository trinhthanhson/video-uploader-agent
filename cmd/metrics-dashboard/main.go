package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"
)

type UploadMetric struct {
	OrderID            string    `json:"order_id"`
	FileName           string    `json:"file_name"`
	FileSizeBytes      int64     `json:"file_size_bytes"`
	DetectedAt         time.Time `json:"detected_at"`
	UploadStartedAt    time.Time `json:"upload_started_at"`
	UploadCompletedAt  time.Time `json:"upload_completed_at"`
	UploadDurationSec  float64   `json:"upload_duration_seconds"`
	TotalProcessingSec float64   `json:"total_processing_seconds"`
	UploadSpeedMBps    float64   `json:"upload_speed_mbps"`
	Status             string    `json:"status"`
	ErrorMessage       string    `json:"error_message,omitempty"`
}

type ChartResponse struct {
	SpeedLabels       []string  `json:"speed_labels"`
	SpeedValues       []float64 `json:"speed_values"`
	ProcessLabels     []string  `json:"process_labels"`
	ProcessValues     []float64 `json:"process_values"`
	HourlyLabels      []string  `json:"hourly_labels"`
	HourlyFileCounts  []int     `json:"hourly_file_counts"`
	TotalFiles        int       `json:"total_files"`
	SuccessFiles      int       `json:"success_files"`
	AverageSpeedMBps  float64   `json:"average_speed_mbps"`
	AverageProcessSec float64   `json:"average_processing_seconds"`
}

const metricsFile = "upload-metrics.jsonl"

func main() {
	http.HandleFunc("/", serveDashboard)
	http.HandleFunc("/api/charts", serveChartData)

	fmt.Println("Metrics dashboard running at http://127.0.0.1:8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>Upload Metrics Dashboard</title>
  <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  <style>
    body {
      font-family: Arial, sans-serif;
      margin: 24px;
      background: #f7f7f7;
    }
    h1 {
      margin-bottom: 8px;
    }
    .summary {
      display: flex;
      gap: 16px;
      margin-bottom: 24px;
      flex-wrap: wrap;
    }
    .card {
      background: white;
      padding: 16px;
      border-radius: 12px;
      min-width: 220px;
      box-shadow: 0 2px 8px rgba(0,0,0,0.08);
    }
    .chart-box {
      background: white;
      padding: 16px;
      border-radius: 12px;
      margin-bottom: 24px;
      box-shadow: 0 2px 8px rgba(0,0,0,0.08);
    }
    canvas {
      max-width: 100%;
      height: 360px !important;
    }
  </style>
</head>
<body>
  <h1>Upload Metrics Dashboard</h1>
  <p>The dashboard reads data from <code>upload-metrics.jsonl</code>.</p>

  <div class="summary">
    <div class="card">
      <h3>Total Files</h3>
      <div id="totalFiles">-</div>
    </div>
    <div class="card">
      <h3>Success Files</h3>
      <div id="successFiles">-</div>
    </div>
    <div class="card">
      <h3>Average Upload Speed</h3>
      <div id="avgSpeed">-</div>
    </div>
    <div class="card">
      <h3>Average Processing Time</h3>
      <div id="avgProcessing">-</div>
    </div>
  </div>

  <div class="chart-box">
    <h3>Upload Speed (MB/s) by File</h3>
    <canvas id="speedChart"></canvas>
  </div>

  <div class="chart-box">
    <h3>Total Processing Time (seconds) by File</h3>
    <canvas id="processingChart"></canvas>
  </div>

  <div class="chart-box">
    <h3>Uploaded Files per Hour</h3>
    <canvas id="hourlyChart"></canvas>
  </div>

  <script>
    async function loadCharts() {
      const res = await fetch('/api/charts');
      const data = await res.json();

      document.getElementById('totalFiles').innerText = data.total_files;
      document.getElementById('successFiles').innerText = data.success_files;
      document.getElementById('avgSpeed').innerText = data.average_speed_mbps.toFixed(2) + ' MB/s';
      document.getElementById('avgProcessing').innerText = data.average_processing_seconds.toFixed(2) + ' s';

      new Chart(document.getElementById('speedChart'), {
        type: 'line',
        data: {
          labels: data.speed_labels,
          datasets: [{
            label: 'Upload Speed (MB/s)',
            data: data.speed_values,
            fill: false,
            tension: 0.2
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false
        }
      });

      new Chart(document.getElementById('processingChart'), {
        type: 'bar',
        data: {
          labels: data.process_labels,
          datasets: [{
            label: 'Processing Time (seconds)',
            data: data.process_values
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false
        }
      });

      new Chart(document.getElementById('hourlyChart'), {
        type: 'bar',
        data: {
          labels: data.hourly_labels,
          datasets: [{
            label: 'Files Uploaded',
            data: data.hourly_file_counts
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false
        }
      });
    }

    loadCharts();
  </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func serveChartData(w http.ResponseWriter, r *http.Request) {
	metrics, err := readMetrics(metricsFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read metrics: %v", err), http.StatusInternalServerError)
		return
	}

	resp := buildChartResponse(metrics)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func readMetrics(path string) ([]UploadMetric, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []UploadMetric{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var metrics []UploadMetric
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var m UploadMetric
		if err := json.Unmarshal(line, &m); err != nil {
			continue
		}

		metrics = append(metrics, m)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].UploadCompletedAt.Before(metrics[j].UploadCompletedAt)
	})

	return metrics, nil
}

func buildChartResponse(metrics []UploadMetric) ChartResponse {
	resp := ChartResponse{}

	hourlyMap := make(map[string]int)
	var totalSpeed float64
	var totalProcessing float64

	for _, m := range metrics {
		resp.TotalFiles++

		if m.Status == "uploaded" {
			resp.SuccessFiles++
			totalSpeed += m.UploadSpeedMBps
			totalProcessing += m.TotalProcessingSec

			resp.SpeedLabels = append(resp.SpeedLabels, m.FileName)
			resp.SpeedValues = append(resp.SpeedValues, m.UploadSpeedMBps)

			resp.ProcessLabels = append(resp.ProcessLabels, m.FileName)
			resp.ProcessValues = append(resp.ProcessValues, m.TotalProcessingSec)

			hourKey := m.UploadCompletedAt.Format("2006-01-02 15:00")
			hourlyMap[hourKey]++
		}
	}

	if resp.SuccessFiles > 0 {
		resp.AverageSpeedMBps = totalSpeed / float64(resp.SuccessFiles)
		resp.AverageProcessSec = totalProcessing / float64(resp.SuccessFiles)
	}

	var hourlyKeys []string
	for k := range hourlyMap {
		hourlyKeys = append(hourlyKeys, k)
	}
	sort.Strings(hourlyKeys)

	for _, k := range hourlyKeys {
		resp.HourlyLabels = append(resp.HourlyLabels, k)
		resp.HourlyFileCounts = append(resp.HourlyFileCounts, hourlyMap[k])
	}

	return resp
}
