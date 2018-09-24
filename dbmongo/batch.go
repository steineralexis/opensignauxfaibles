package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// AdminBatch metadata Batch
type AdminBatch struct {
	ID       AdminID    `json:"id" bson:"_id"`
	Files    BatchFiles `json:"files" bson:"files"`
	Readonly bool       `json:"readonly" bson:"readonly"`
	Params   struct {
		DateDebut       time.Time `json:"date_debut" bson:"date_debut"`
		DateFin         time.Time `json:"date_fin" bson:"date_fin"`
		DateFinEffectif time.Time `json:"date_fin_effectif" bson:"date_fin_effectif"`
	} `json:"params" bson:"param"`
	ChanEtablissement chan *ValueEtablissement `json:"-" bson:"-"`
	ChanEntreprise    chan *ValueEntreprise    `json:"-" bson:"-"`
}

// BatchFiles fichiers mappés par type
type BatchFiles map[string][]string

func (batchFiles BatchFiles) attachFile(fileType string, file string) {
	batchFiles[fileType] = append(batchFiles[fileType], file)
}

func isBatchID(batchID string) bool {
	_, err := time.Parse("0601", batchID)
	return err == nil
}

func (batch *AdminBatch) load(
	batchKey string,
	db *mgo.Database,
	chanEtablissement chan *ValueEtablissement,
	chanEntreprise chan *ValueEntreprise) error {
	err := db.C("Admin").Find(bson.M{"_id.type": "batch", "_id.key": batchKey}).One(batch)
	batch.ChanEntreprise = chanEntreprise
	batch.ChanEtablissement = chanEtablissement
	return err
}

func (batch *AdminBatch) save(db *mgo.Database) error {
	_, err := db.C("Admin").Upsert(bson.M{"_id": batch.ID}, batch)
	return err
}

func (batch *AdminBatch) new(batchID string) error {
	if batchID == "" {
		return errors.New("Valeur de batch non autorisée")
	}
	batch.ID.Key = batchID
	batch.ID.Type = "batch"
	batch.Files = BatchFiles{}
	return nil
}

func nextBatchID(batchID string) (string, error) {
	batchTime, err := time.Parse("0601", batchID)
	if err != nil {
		return "", err
	}
	nextBatchTime := time.Date(batchTime.Year(), time.Month(batchTime.Month()+1), 1, 0, 0, 0, 0, time.UTC)
	return nextBatchTime.Format("0601"), err
}

func sp(s string) *string {
	return &s
}

func upsertBatch(c *gin.Context) {
	status := c.Keys["status"].(*DBStatus)
	// err := status.setDBStatus(sp("Sauvegarde du Batch"))
	// if err != nil {
	// 	c.JSON(500, err)
	// 	fmt.Println(err)
	// 	return
	// }

	db := c.Keys["db"].(*mgo.Database)
	batch := AdminBatch{}
	err := c.Bind(&batch)
	if err != nil {
		c.JSON(500, err)
		fmt.Println(err)
		return
	}

	err = batch.save(db)
	if err != nil {
		c.JSON(500, "Erreur à l'enregistrement")
		fmt.Println(err)
		return
	}

	status.Epoch++
	status.write()

	c.JSON(200, batch)
}

func listBatch(c *gin.Context) {
	db := c.Keys["db"].(*mgo.Database)
	var batch []AdminBatch
	err := db.C("Admin").Find(bson.M{"_id.type": "batch"}).Sort("-_id.key").All(&batch)
	if err != nil {
		spew.Dump(err)
		c.JSON(500, err)
		return
	}
	c.JSON(200, batch)
}

func getBatchesID(db *mgo.Database) []string {
	batches := getBatches(db)
	var batchesID []string
	for _, b := range batches {
		batchesID = append(batchesID, b.ID.Key)
	}
	return batchesID
}

func getBatches(db *mgo.Database) []*AdminBatch {
	var batches []*AdminBatch
	db.C("Admin").Find(bson.M{"_id.type": "batch"}).Sort("_id.key").All(&batches)
	return batches
}

func getBatch(db *mgo.Database, batchID string) (*AdminBatch, error) {
	var batch AdminBatch
	err := db.C("Admin").Find(bson.M{"_id.type": "batch", "_id.key": batchID}).One(&batch)
	spew.Dump(batch)
	return &batch, err
}

// batchToTime calcule la date de référence à partir de la référence de batch
func batchToTime(batch string) (time.Time, error) {
	year, err := strconv.Atoi(batch[0:2])
	if err != nil {
		return time.Time{}, err
	}

	month, err := strconv.Atoi(batch[2:4])
	if err != nil {
		return time.Time{}, err
	}

	date := time.Date(2000+year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return date, err
}

func processBatch(c *gin.Context) {
	dbstatus := c.Keys["status"].(*DBStatus)

	go func() {
		dbstatus.setDBStatus(sp("Import des fichiers"))
		time.Sleep(10 * time.Second)
		dbstatus.setDBStatus(nil)
		// - compact
		message := "Compactage des données"
		dbstatus.setDBStatus(&message)
		time.Sleep(5 * time.Second)
		dbstatus.setDBStatus(nil)
		// - reduce
		message = "Calcul des variables"
		dbstatus.setDBStatus(&message)
		dbstatus.setDBStatus(nil)
		// - predict
		message = "Calcul de la prédiction"
		dbstatus.setDBStatus(&message)
		time.Sleep(5 * time.Second)
		dbstatus.setDBStatus(nil)
		// - createNextBatch
		message = "Clôture du batch et initialisation du suivant"
		dbstatus.setDBStatus(&message)
		createNextBatch(c)
		dbstatus.setDBStatus(nil)
	}()
	c.JSON(200, "ok !")
}

func lastBatch(db *mgo.Database) string {
	batches := getBatches(db)
	l := len(batches)
	batch := batches[l-1]
	return batch.ID.Key
}

func createNextBatch(c *gin.Context) {
	db := c.Keys["db"].(*mgo.Database)
	batchID, _ := nextBatchID(lastBatch(db))
	batch := AdminBatch{
		ID: AdminID{
			Key:  batchID,
			Type: "batch",
		},
	}
	err := batch.save(db)
	if err != nil {
		c.JSON(500, err)
	} else {
		c.JSON(200, err)
	}
}
