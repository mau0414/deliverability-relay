package api

import (
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/mau0414/deliverability-relay/internal/domain"
	"github.com/mau0414/deliverability-relay/internal/repository"
)

const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>MTA Dashboard</title>
	<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.4/dist/chart.umd.min.js"></script>
	<style>
		body { font-family: system-ui, sans-serif; margin: 40px; background: #0f0f0f; color: #eee; }
		h1 { font-size: 1.5rem; }
		h2 { font-size: 1.1rem; color: #ccc; margin-top: 40px; }
		.cards { display: flex; gap: 20px; margin: 30px 0; flex-wrap: wrap; }
		.card { background: #1a1a1a; padding: 20px 24px; border-radius: 8px; min-width: 140px; }
		.card .value { font-size: 2rem; font-weight: bold; }
		.card .label { color: #999; font-size: 0.85rem; text-transform: uppercase; letter-spacing: 0.05em; }
		table { border-collapse: collapse; width: 100%; margin-top: 12px; }
		th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #333; }
		th { color: #999; font-weight: normal; font-size: 0.85rem; text-transform: uppercase; }
		.chart-row { display: flex; gap: 20px; flex-wrap: wrap; }
		.chart-container { background: #1a1a1a; border-radius: 8px; padding: 20px; margin-top: 12px; }
	</style>
</head>
<body>
	<h1>MTA Dashboard</h1>

	<div class="cards">
		<div class="card">
			<div class="value">{{.Total}}</div>
			<div class="label">Total emails</div>
		</div>
		<div class="card">
			<div class="value">{{printf "%.1f" .DeliveryRate}}%</div>
			<div class="label">Delivery rate</div>
		</div>
		<div class="card">
			<div class="value">{{printf "%.1f" .BounceRate}}%</div>
			<div class="label">Bounce rate</div>
		</div>
	</div>

	<h2>Volume by day</h2>
	<div class="chart-row">
		<div class="chart-container" style="flex: 2; min-width: 400px;">
			<canvas id="volumeChart"></canvas>
		</div>
		<div class="chart-container" style="flex: 1; min-width: 280px;">
			<canvas id="statusChart"></canvas>
		</div>
	</div>

	<h2>By status</h2>
	<table>
		<tr><th>Status</th><th>Count</th></tr>
		{{range .ByStatus}}
		<tr><td>{{.Status}}</td><td>{{.Count}}</td></tr>
		{{end}}
	</table>

	<script>
		const labels = {{.DailyLabels}};
		const values = {{.DailyValues}};

		new Chart(document.getElementById('volumeChart'), {
			type: 'line',
			data: {
				labels: labels,
				datasets: [{
					label: 'Emails per day',
					data: values,
					borderColor: '#4a9eff',
					backgroundColor: 'rgba(74, 158, 255, 0.15)',
					fill: true,
					tension: 0.3,
					pointRadius: 4,
				}]
			},
			options: {
				responsive: true,
				plugins: { legend: { display: false } },
				scales: {
					x: { ticks: { color: '#999' }, grid: { color: '#2a2a2a' } },
					y: { ticks: { color: '#999' }, grid: { color: '#2a2a2a' }, beginAtZero: true }
				}
			}
		});

		const statusLabels = {{.StatusLabels}};
		const statusValues = {{.StatusValues}};

		const statusColors = {
			sent: '#4ade80',
			bounced: '#f87171',
			failed: '#fb923c',
			queued: '#60a5fa',
			sending: '#c084fc',
		};

		new Chart(document.getElementById('statusChart'), {
			type: 'doughnut',
			data: {
				labels: statusLabels,
				datasets: [{
					data: statusValues,
					backgroundColor: statusLabels.map(s => statusColors[s] || '#999'),
					borderColor: '#1a1a1a',
					borderWidth: 2,
				}]
			},
			options: {
				responsive: true,
				plugins: {
					legend: { position: 'bottom', labels: { color: '#ccc' } }
				}
			}
		});
	</script>
</body>
</html>
`

type dashboardViewData struct {
	Total        int
	DeliveryRate float64
	BounceRate   float64
	ByStatus     []repository.StatusCount
	DailyLabels  template.JS
	DailyValues  template.JS
	StatusLabels template.JS
	StatusValues template.JS
}

func (s *Server) handleDashboard() http.HandlerFunc {
	tmpl := template.Must(template.New("dashboard").Parse(dashboardHTML))

	return func(w http.ResponseWriter, r *http.Request) {
		metrics, err := s.repository.GetMetrics(r.Context())
		if err != nil {
			http.Error(w, "failed to load metrics", http.StatusInternalServerError)
			return
		}

		view := dashboardViewData{
			ByStatus: metrics.ByStatus,
		}

		var sent, bounced int
		for _, sc := range metrics.ByStatus {
			view.Total += sc.Count
			switch sc.Status {
			case domain.StatusSent:
				sent = sc.Count
			case domain.StatusBounced:
				bounced = sc.Count
			}
		}

		if view.Total > 0 {
			view.DeliveryRate = float64(sent) / float64(view.Total) * 100
			view.BounceRate = float64(bounced) / float64(view.Total) * 100
		}

		labels := make([]string, len(metrics.Daily))
		values := make([]int, len(metrics.Daily))
		for i, d := range metrics.Daily {
			labels[i] = d.Date.Format("2006-01-02")
			values[i] = d.Count
		}

		labelsJSON, err := json.Marshal(labels)
		if err != nil {
			http.Error(w, "failed to render chart data", http.StatusInternalServerError)
			return
		}
		valuesJSON, err := json.Marshal(values)
		if err != nil {
			http.Error(w, "failed to render chart data", http.StatusInternalServerError)
			return
		}

		view.DailyLabels = template.JS(labelsJSON)
		view.DailyValues = template.JS(valuesJSON)

		statusLabels := make([]string, len(metrics.ByStatus))
		statusValues := make([]int, len(metrics.ByStatus))
		for i, sc := range metrics.ByStatus {
			statusLabels[i] = string(sc.Status)
			statusValues[i] = sc.Count
		}

		statusLabelsJSON, err := json.Marshal(statusLabels)
		if err != nil {
			http.Error(w, "failed to render chart data", http.StatusInternalServerError)
			return
		}
		statusValuesJSON, err := json.Marshal(statusValues)
		if err != nil {
			http.Error(w, "failed to render chart data", http.StatusInternalServerError)
			return
		}

		view.StatusLabels = template.JS(statusLabelsJSON)
		view.StatusValues = template.JS(statusValuesJSON)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, view)
	}
}