// Copyright 2026 LogicalClocks AB
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"context"
	"testing"
	"time"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var tableMemoryColumns = []string{"database_name", "table_name", "in_memory_bytes", "free_in_memory_bytes",
	"disk_memory_bytes", "free_disk_memory_bytes"}

func TestTableMemoryUsageIsScannedOncePerInterval(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()
	tableMemoryUsageCache.rows = nil

	mock.ExpectQuery(sanitizeQuery(ndbinfoTableMemoryUsageQuery)).WillReturnRows(
		sqlmock.NewRows(tableMemoryColumns).AddRow("proj", "fg_1", 100, 1, 0, 0))
	mock.ExpectQuery(sanitizeQuery(ndbinfoTableMemoryUsageQuery)).WillReturnRows(
		sqlmock.NewRows(tableMemoryColumns).AddRow("proj", "fg_1", 200, 1, 0, 0))

	start := time.Now()
	for _, offset := range []time.Duration{0, 30 * time.Second, 59 * time.Second} {
		rows, err := tableMemoryUsageRows(context.Background(), db, time.Minute, start.Add(offset))
		if err != nil || len(rows) != 1 || rows[0].inMemoryBytes != 100 {
			t.Fatalf("at +%s: rows %v, err %v; want the first scan", offset, rows, err)
		}
	}
	rows, err := tableMemoryUsageRows(context.Background(), db, time.Minute, start.Add(time.Minute))
	if err != nil || rows[0].inMemoryBytes != 200 {
		t.Fatalf("after the interval: rows %v, err %v; want a new scan", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}

func TestTableMemoryUsageWithoutIntervalScansEveryTime(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()
	tableMemoryUsageCache.rows = nil

	for i := 0; i < 2; i++ {
		mock.ExpectQuery(sanitizeQuery(ndbinfoTableMemoryUsageQuery)).WillReturnRows(
			sqlmock.NewRows(tableMemoryColumns).AddRow("proj", "fg_1", 100, 1, 0, 0))
	}
	now := time.Now()
	for i := 0; i < 2; i++ {
		if _, err := tableMemoryUsageRows(context.Background(), db, 0, now); err != nil {
			t.Fatalf("scan %d: %s", i, err)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled exceptions: %s", err)
	}
}
