package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func serveAdmin() error {
	return http.ListenAndServe(_slaveAdminPort, makeAdminHandler())
}

func makeAdminHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			arr := make([]string, len(_localServices))
			i := 0
			for k := range _localServices {
				arr[i] = k
				i++
			}
			byt, _ := json.Marshal(arr)
			w.Write(byt)

		case http.MethodPost:

			byt, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "couldn't read body", http.StatusInternalServerError)
				return
			}
			r.Body.Close()

			localRequest := Request{}

			err = json.Unmarshal(byt, &localRequest)
			if err != nil {
				http.Error(w, "couldn't unmarshal body", http.StatusInternalServerError)
				return
			}

			res, err := register(localRequest)
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprintf(w, "error registering service %s\n%s", localRequest.Name, err.Error())
				return
			}


			w.WriteHeader(http.StatusOK)
			w.Write(res)
		}
	}
}
