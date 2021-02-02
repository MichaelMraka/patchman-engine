package evaluator

import (
	"app/base/database"
	"app/base/mqueue"
	"context"
	"github.com/RedHatInsights/patchman-clients/vmaas"
	"github.com/pkg/errors"
)

func makeVirtualVmaasResp(event mqueue.PlatformEvent) (*vmaas.UpdatesV2Response, error) {
	var res vmaas.UpdatesV2Response
	for _, up := range event.Updates {
		packageUpdates := res.UpdateList[up.Package]
		packageUpdates.AvailableUpdates = append(packageUpdates.AvailableUpdates, vmaas.UpdatesResponseAvailableUpdates{
			Repository: "",
			Releasever: "",
			Basearch:   "",
			Erratum:    up.Advisory,
			Package:    up.Package,
		})
		res.UpdateList[up.Package] = packageUpdates
	}
	return &res, nil
}

func EvaluateClientside(ctx context.Context, event mqueue.PlatformEvent) error {
	vmaasResp, err := makeVirtualVmaasResp(event)
	if err != nil {
		return err
	}
	tx := database.Db.BeginTx(ctx, nil)
	// Don'requested allow TX to hang around locking the rows
	defer tx.RollbackUnlessCommitted()

	// TODO: Replace the account id
	system, err := loadSystemData(tx, 666, event.ID)
	if err != nil {
		evaluationCnt.WithLabelValues("error-db-read-inventory-data").Inc()
		return nil
	}
	err = evaluateAndStore(tx, system, *vmaasResp)
	if err != nil {
		return errors.Wrap(err, "Unable to evaluate and store results")
	}

	err = commitWithObserve(tx)
	if err != nil {
		evaluationCnt.WithLabelValues("error-database-commit").Inc()
		return errors.New("database commit failed")
	}

	err = publishRemediationsState(system.InventoryID, *vmaasResp)
	if err != nil {
		evaluationCnt.WithLabelValues("error-remediations-publish").Inc()
		return errors.Wrap(err, "remediations publish failed")
	}

	evaluationCnt.WithLabelValues("success").Inc()
	return nil
}
