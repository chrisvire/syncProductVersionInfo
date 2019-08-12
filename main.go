package main

import (
	"database/sql"
	"fmt"
	"github.com/hashicorp/go-version"
	_ "github.com/lib/pq"
	"github.com/satori/go.uuid"
	"strings"
)

type ProductWithBuildVersion struct {
	Id uuid.UUID `sql:",type:uuid"`
	Version string
	BuildId int
}

type ProductBuildVersion struct {
	Id int
	Version string
}

type ProductVersion struct {
	Id uuid.UUID `sql:",type:uuid"`
	Version string
}

func portalDbConn() (db *sql.DB) {
	connStr := "postgres://appbuilder:PASSWORD@aps-portal-prd.cdiwgmpebjk2.us-east-1.rds.amazonaws.com/portal"
	//connStr := "postgres://appbuilder:PASSWORD@aps-portal-stg.cdiwgmpebjk2.us-east-1.rds.amazonaws.com/portal"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func getProductsWithLatestSuccessfulBuild(db *sql.DB) map[int]*ProductWithBuildVersion {
	selDB, err := db.Query(`select "Products"."Id", "ProductBuilds"."Version", "ProductBuilds"."Id" from "Products", "ProductBuilds" where "Products"."Id" = "ProductBuilds"."ProductId" and "ProductBuilds"."Version" notnull`)
	if err != nil {
		panic(err.Error())
	}
	defer selDB.Close()

	products := map[int]*ProductWithBuildVersion{}
	for selDB.Next() {
		var id uuid.UUID
		var version string
		var buildId int

		err = selDB.Scan(&id, &version, &buildId)
		product := &ProductWithBuildVersion{Id:id, Version:version, BuildId:buildId}
		if err != nil {
			panic(err.Error())
		}

		products[buildId]=product
	}
	return products
}

func getProductBuildVersions(db *sql.DB) map[int]*ProductBuildVersion {
	selDB, err := db.Query(`select "Id", "Version" from "ProductBuilds" where "ProductBuilds"."Version" notnull`)
	if err != nil {
		panic(err.Error())
	}
	defer selDB.Close()

	productBuildVersions := map[int]*ProductBuildVersion{}
	for selDB.Next() {
		var id int
		var version string

		err = selDB.Scan(&id, &version)
		productBuildVersion := &ProductBuildVersion{Id: id, Version:version}
		if err != nil {
			panic(err.Error())
		}

		productBuildVersions[id]= productBuildVersion
	}
	return productBuildVersions
}

func getProductVersions(db *sql.DB) map[uuid.UUID]*ProductVersion {
	selDB, err := db.Query(`select "Id", "VersionBuilt" from "Products" where "Products"."VersionBuilt" notnull`)
	if err != nil {
		panic(err.Error())
	}
	defer selDB.Close()

	productVersions := map[uuid.UUID]*ProductVersion{}
	for selDB.Next() {
		var id uuid.UUID
		var version string

		err = selDB.Scan(&id, &version)
		productVersion := &ProductVersion{Id: id, Version:version}
		if err != nil {
			panic(err.Error())
		}

		productVersions[id]= productVersion
	}
	return productVersions
}

func parseVersion(productVersion string) *version.Version {
	strVer := "0.0"
	fields := strings.Fields(productVersion)
	if len(fields) == 1 {
		strVer = fields[0]
	} else if len(fields) > 1 {
		verName := fields[0]
		build := strings.Trim(fields[1], "()")
		strVer = fmt.Sprintf("%s.%s", verName, build)
	}

   	ver, err := version.NewVersion(strVer)
	if err != nil {
		panic(err.Error())
	}
	return ver
}

func parseVersionNameAndCode(productVersion string) (string, string) {
	ver := parseVersion(productVersion)
	verSegs := ver.Segments()
	verName := strings.Trim(strings.Join(strings.Split(fmt.Sprint(verSegs[0:len(verSegs)-1]), " "), "."), "[]")
	verCode := verSegs[len(verSegs)-1]
    return verName, fmt.Sprintf("%d", verCode)
}


func setProductVersionBuilt(portalDB *sql.DB) {
	products := getProductsWithLatestSuccessfulBuild(portalDB)
	productVersions := map[uuid.UUID]string{}
	for _, product := range products {
		if val, ok := productVersions[product.Id]; ok {
			curVer := parseVersion(val)
			newVer := parseVersion(product.Version)
			if newVer.GreaterThan(curVer) {
				productVersions[product.Id] = product.Version
			}
		} else {
			productVersions[product.Id] = product.Version
		}
	}
	fmt.Println("----------")
	for id, version := range productVersions {
		fmt.Printf(`UPDATE "Products" SET "VersionBuilt" = '%s' WHERE "Id"::text = '%s';`, version, id)
		fmt.Println()
	}
}

func separateBuildNumberFromVersion(portalDB *sql.DB) {
	productBuildVersions := getProductBuildVersions(portalDB)
	for id, productBuildVersion := range productBuildVersions {
		if !strings.Contains(productBuildVersion.Version, "(") {
			verName, verCode := parseVersionNameAndCode(productBuildVersion.Version)
			fmt.Printf(`UPDATE "ProductBuilds" SET "Version" = '%s (%s)' WHERE "Id" = %d;`, verName, verCode, id)
			fmt.Println()
		}
	}

	productVersions := getProductVersions(portalDB)
	for id, productVersion := range productVersions {
		if !strings.Contains(productVersion.Version, "(") {
			verName, verCode := parseVersionNameAndCode(productVersion.Version)
			fmt.Printf(`UPDATE "Products" SET "VersionBuilt" = '%s (%s)' WHERE "Id"::text = '%s';`, verName, verCode, id)
			fmt.Println()
		}
	}
}

func main() {
	portalDB := portalDbConn()
	defer portalDB.Close()

	//setProductVersionBuilt(portalDB)
	separateBuildNumberFromVersion(portalDB)
}