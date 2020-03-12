package indexer

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

type databaseWriter struct {
	dbWriter *elasticsearch.Client
}

func newDatabaseWriter(cfg elasticsearch.Config) (*databaseWriter, error) {
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &databaseWriter{dbWriter: es}, nil
}

// CheckAndCreateIndex will check if a index exits and if dont will create a new one
func (dw *databaseWriter) CheckAndCreateIndex(index string, body io.Reader) error {
	var res *esapi.Response
	var err error
	defer func() {
		closeESResponseBody(res)
	}()

	res, err = dw.dbWriter.Indices.Exists([]string{index})
	if err != nil {
		return err
	}

	// Indices.Exists actually does a HEAD request to the elastic index.
	// A status code of 200 actually means the index exists so we
	//  don't need to do anything.
	if res.StatusCode == http.StatusOK {
		return nil
	}
	// A status code of 404 means the index does not exist so we create it
	if res.StatusCode == http.StatusNotFound {
		err = dw.createDatabaseIndex(index, body)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dw *databaseWriter) createDatabaseIndex(index string, body io.Reader) error {
	var err error
	var res *esapi.Response
	defer func() {
		closeESResponseBody(res)
	}()

	if body != nil {
		res, err = dw.dbWriter.Indices.Create(index, dw.dbWriter.Indices.Create.WithBody(body))
	} else {
		res, err = dw.dbWriter.Indices.Create(index)
	}

	if err != nil {
		return err
	}

	if res.IsError() {
		// Resource already exists
		if res.StatusCode == http.StatusBadRequest {
			return nil
		}

		log.Warn("indexer: resource already exists", "error", res.String())
		return ErrCannotCreateIndex
	}

	return nil
}

// DoRequest will do a request to elastic server
func (dw *databaseWriter) DoRequest(req *esapi.IndexRequest) error {
	var err error
	var res *esapi.Response
	defer func() {
		closeESResponseBody(res)
	}()

	res, err = req.Do(context.Background(), dw.dbWriter)
	if err != nil {
		return err
	}

	if res.IsError() {
		log.Warn("indexer", "error", res.String())
	}

	return nil
}

// DoBulkRequest will do a bulk of request to elastic server
func (dw *databaseWriter) DoBulkRequest(buff *bytes.Buffer, index string) error {
	reader := bytes.NewReader(buff.Bytes())

	var err error
	var res *esapi.Response
	defer func() {
		closeESResponseBody(res)
	}()

	res, err = dw.dbWriter.Bulk(reader, dw.dbWriter.Bulk.WithIndex(index))
	if err != nil {
		return err
	}

	if res.IsError() {
		log.Warn("indexer", "error", res.String())
	}

	return nil
}

func closeESResponseBody(res *esapi.Response) {
	if res != nil && res.Body != nil {
		err := res.Body.Close()
		if err != nil {
			log.Trace("error closing elastic search response body", "error", err)
		}
	}
}
