package operator

import (
	"encoding/json"
	queueentryoperatorBetav1 "github.com/podnov/k8s-queue-entry-operator/pkg/apis/queueentryoperator/betav1"
	"github.com/podnov/k8s-queue-entry-operator/pkg/internal"
	"testing"
)

func Test_diffObjects_changed_false(t *testing.T) {
	givenBytes, err := internal.ReadRelativeFile("operator-utils_test_Test_diffObjects_changed_false_given.json")
	if err != nil {
		t.Error(err)
	}

	givenOld := queueentryoperatorBetav1.DbQueue{}
	err = json.Unmarshal(givenBytes, &givenOld)
	if err != nil {
		t.Error(err)
	}

	givenNew := queueentryoperatorBetav1.DbQueue{}
	err = json.Unmarshal(givenBytes, &givenNew)
	if err != nil {
		t.Error(err)
	}

	actual := diffObjects(givenOld, givenNew)

	if actual != false {
		t.Errorf("got: %v", actual)
	}
}

func Test_diffObjects_changed_true(t *testing.T) {
	givenOldBytes, err := internal.ReadRelativeFile("operator-utils_test_Test_diffObjects_changed_true_givenOld.json")
	if err != nil {
		t.Error(err)
	}
	givenNewBytes, err := internal.ReadRelativeFile("operator-utils_test_Test_diffObjects_changed_true_givenNew.json")
	if err != nil {
		t.Error(err)
	}

	givenOld := queueentryoperatorBetav1.DbQueue{}
	err = json.Unmarshal(givenOldBytes, &givenOld)
	if err != nil {
		t.Error(err)
	}

	givenNew := queueentryoperatorBetav1.DbQueue{}
	err = json.Unmarshal(givenNewBytes, &givenNew)
	if err != nil {
		t.Error(err)
	}

	actual := diffObjects(givenOld, givenNew)

	if actual != true {
		t.Errorf("got: %v", actual)
	}
}
