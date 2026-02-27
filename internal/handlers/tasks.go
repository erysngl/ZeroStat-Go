package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/erysngl/zerostat/internal/process"
)

func ServeTasks(w http.ResponseWriter, r *http.Request) {
	data := getBaseData()
	// base.html requires .Data to not be nil to render the navbar
	data.Data = struct{}{}
	tmplCache["tasks.html"].ExecuteTemplate(w, "base.html", data)
}

func ServeTasksList(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	sortBy := r.FormValue("sort_by")
	if sortBy == "" {
		sortBy = "cpu"
	}
	sortDir := r.FormValue("sort_dir")
	if sortDir == "" {
		sortDir = "desc"
	}

	procs := process.GetProcesses(query, sortBy, sortDir)

	pageStr := r.FormValue("page")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	limit := 15
	total := len(procs)
	totalPages := (total + limit - 1) / limit

	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	start := (page - 1) * limit
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > total {
		end = total
	}

	var slicedProcs []process.ProcessInfo
	if start < total {
		slicedProcs = procs[start:end]
	}

	var pages []int
	startPage := page - 2
	if startPage < 1 {
		startPage = 1
	}
	endPage := startPage + 4
	if endPage > totalPages {
		endPage = totalPages
		startPage = endPage - 4
		if startPage < 1 {
			startPage = 1
		}
	}
	for i := startPage; i <= endPage; i++ {
		pages = append(pages, i)
	}

	data := getBaseData()
	data.Data = struct {
		Processes  []process.ProcessInfo
		Query      string
		SortBy     string
		SortDir    string
		Page       int
		TotalPages int
		Pages      []int
		HasPrev    bool
		HasNext    bool
		PrevPage   int
		NextPage   int
	}{
		Processes:  slicedProcs,
		Query:      query,
		SortBy:     sortBy,
		SortDir:    sortDir,
		Page:       page,
		TotalPages: totalPages,
		Pages:      pages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
	}

	// Return just the table rows partial
	tmplCache["tasks.html"].ExecuteTemplate(w, "tasks_list", data)
}

func HandleKillProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	pidStr := r.FormValue("pid")
	pid, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid PID", http.StatusBadRequest)
		return
	}

	if err := process.KillProcess(int32(pid)); err != nil {
		fmt.Fprintf(w, "<div class='text-red-500 text-sm'>Kill Failed: %v</div>", err)
		return
	}

	// Wait briefly for process to die before refreshing list
	// Normally htms swaps out the row, or we can trigger a list refresh.
	// For simplicity, we can trigger HX-Trigger header to refresh list.
	w.Header().Set("HX-Trigger", "refreshTasks")
	fmt.Fprintf(w, "<div class='text-green-500 text-sm'>Signal sent to PID %d</div>", pid)
}

func HandleStopContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "Invalid Container ID", http.StatusBadRequest)
		return
	}

	if err := process.StopContainer(id); err != nil {
		fmt.Fprintf(w, "<div class='text-red-500 text-sm'>Stop Failed: %v</div>", err)
		return
	}

	w.Header().Set("HX-Trigger", "refreshTasks")
	fmt.Fprintf(w, "<div class='text-green-500 text-sm'>Command executed for container %s</div>", id[:12])
}
