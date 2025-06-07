package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

var publicDimensions [100]Dimension
var newDimension bool

func memoryBuffedRetriever(index int, numDims int) []Dimension {
	end := index + numDims
	if end <= len(publicDimensions) {
		return publicDimensions[index:end]

	} else {
		return sqlRetrieve(index, numDims)
	}
}

// Get
func publicDimensionsReq(w http.ResponseWriter, r *http.Request) {
	querys := r.URL.Query()
	index, err := strconv.Atoi(querys.Get("index"))
	if err != nil {
		http.Error(w, "index query not a valid number", http.StatusBadRequest)
	}
	if index < 0 {
		http.Error(w, "index is negative", http.StatusBadRequest)
	}

	numDims, err := strconv.Atoi(querys.Get("numDims"))
	if err != nil {
		http.Error(w, "numDims query not a valid number", http.StatusBadRequest)
		return
	}
	if numDims < 1 || numDims > 100 {
		http.Error(w, "requested page size is invalid", http.StatusBadRequest)
		return
	}

	dims := memoryBuffedRetriever(index, numDims)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dims)
	logger.Info().Str("ip", GetIP(r)).Int("index", index).Int("pageSize", numDims).Msg("public list request")
}

func sqlRetrieve(index int, numdims int) []Dimension {
	query := `
	select id, title, encrypted, fileName,
	text, reads, downloadLimit, expirationDate
	from space
	Limit ? Offset ?
	`
	rows, err := db.Query(query, numdims, index)
	if err != nil {
		logger.Fatal().Err(err).Msg("fatal error, retrieving public dimensions")
	}
	defer rows.Close()

	var dimensions []Dimension
	for rows.Next() {
		var dim Dimension
		err = rows.Scan(
			&dim.Id, &dim.Title, &dim.Encrypted,
			&dim.FileName, &dim.Text, &dim.Reads,
			&dim.DownloadLimit, &dim.ExpirationDate)
		if err != nil {
			logger.Fatal().Err(err).Msg("fatal error, retrieving public dimension")
		}

		dim.ExpirationDateISO = prettyDate(dim.ExpirationDate)
		dimensions = append(dimensions, dim)
	}
	return dimensions
}

func publicHandler() {
	for {
		if newDimension {
			buffer := sqlRetrieve(0, 99)
			copy(publicDimensions[:], buffer[:])
			newDimension = false
		}
		time.Sleep(500 * time.Millisecond)
	}
}
