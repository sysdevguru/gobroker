package files

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

type SODFile interface {
	ExtCode() string
	Extension() string
	Delimiter() string
	Header() bool
	Value() reflect.Value
	Append(v interface{})
	Sync(asOf time.Time) (uint, uint)
}

// Parse parses a given SODFile from the raw byte array
func Parse(b []byte, file SODFile) error {
	r := bufio.NewReader(bytes.NewReader(b))
	if file.Header() {
		_, _, err := r.ReadLine()
		if err != nil {
			log.Error("sod file parse error", "file", file.ExtCode(), "error", err)
			return err
		}
	}
	start := clock.Now()
	rows, err := Unmarshal(r, &file)
	elapsed := clock.Now().Sub(start)
	if err != nil {
		log.Error("sod file parse error", "file", file.ExtCode(), "error", err)
		return err
	}
	log.Info(
		"sod file parsed",
		"file", file.ExtCode(),
		"rows", rows,
		"elapsed", elapsed,
		"rows/sec", float64(rows)/elapsed.Seconds(),
	)
	return nil
}

func Unmarshal(r *bufio.Reader, v *SODFile) (rows int, err error) {
	sl := (*v).Value()
	st := sl.Type().Elem()

	for {
		b, _, err := r.ReadLine()
		if err != nil {
			// file is finished
			if err == io.EOF {
				return rows, nil
			}
			return 0, err
		}

		if len(b) == 0 {
			return rows, nil
		}

		if (*v).ExtCode() == "EXT001" {
			(*v).Append(string(b))
			rows++
			continue
		}

		record := strings.Split(string(b), (*v).Delimiter())
		newRow := reflect.New(st).Elem()

		if newRow.NumField() != len(record) {
			return 0, fmt.Errorf(
				"%v CSV field mismatch %v : %v",
				(*v).ExtCode(),
				newRow.NumField(),
				len(record),
			)
		}

		for i := 0; i < newRow.NumField(); i++ {
			f := newRow.Field(i)
			if newRow.Type().Field(i).Tag.Get("csv") == "skip" {
				continue
			}
			if record[i] == "" {
				continue
			}
			switch f.Type().String() {
			case "int":
				iVal, err := strconv.ParseInt(record[i], 10, 0)
				if err != nil {
					return 0, err
				}
				f.SetInt(iVal)
			case "decimal.Decimal":
				d, err := decimal.NewFromString(strings.Replace(record[i], "$", "", 1))
				if err != nil {
					return 0, fmt.Errorf("Invalid decimal string %v", record[i])
				}
				f.Set(reflect.ValueOf(d))
			case "*decimal.Decimal":
				d, err := decimal.NewFromString(strings.Replace(record[i], "$", "", 1))
				if err != nil {
					return 0, fmt.Errorf("Invalid decimal string %v", record[i])
				}
				f.Set(reflect.ValueOf(&d))
			case "*string":
				f.Set(reflect.ValueOf(&record[i]))
			default:
				f.Set(reflect.ValueOf(record[i]).Convert(f.Type()))
			}
		}
		(*v).Append(newRow.Interface())
		rows++
	}
}

// StoreErrors stores the batch processing errors reported
// by the individual SoD file Sync() methods.
func StoreErrors(errors []models.BatchError) {
	if len(errors) > 0 {
		tx := db.Begin()
		for _, err := range errors {
			if dbErr := tx.FirstOrCreate(&err).Error; dbErr != nil {
				log.Error("sod file error storage failure", "error", dbErr)
			}
		}
		tx.Commit()
	}
}

// IsFirmAccount identifies firm accounts that are not to be
// processed during the start of day batch processing
func IsFirmAccount(apexAcct string) bool {
	if utils.Prod() {
		switch apexAcct {
		// error account
		case "3AP00101":
			fallthrough
		// client test account 1
		case "3AP03901":
			fallthrough
		// client test account 2
		case "3AP03902":
			fallthrough
		// firm test account
		case "3AP00002":
			fallthrough
		// reward account
		case "3AP00102":
			fallthrough
		// deposit account
		case "3AP00100":
			return true
		}
	}
	return false
}
