package controllers

import (
	"app/base/database"
	"app/base/utils"
	"app/manager/middlewares"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"net/http"
)

var PackageSystemFields = database.MustGetQueryAttrs(&PackageSystemItem{})
var PackageSystemsSelect = database.MustGetSelect(&PackageSystemItem{})
var PackageSystemsOpts = ListOpts{
	Fields: PackageSystemFields,
	// By default, we show only fresh systems. If all systems are required, you must pass in:true,false filter into the api
	DefaultFilters: map[string]FilterData{},
	DefaultSort:    "id",
}

type PackageSystemItem struct {
	ID            string `json:"id" query:"sp.inventory_id"`
	InstalledEVRA string `json:"installed_evra" query:"p.evra"`
	AvailableEVRA string `json:"available_evra" query:"spkg.latest_evra"`
	Updatable     bool   `json:"updatable" query:"spkg.latest_evra IS NOT NULL"`
}

type PackageSystemsResponse struct {
	Data  []PackageSystemItem `json:"data"`
	Links Links               `json:"links"`
	Meta  ListMeta            `json:"meta"`
}

func packageSystemsQuery(acc int, pkgName string) *gorm.DB {
	return database.Db.
		Select(PackageSystemsSelect).
		Table("system_platform sp").
		// nolint: lll
		Joins("inner join system_package spkg on spkg.system_id = sp.id and sp.stale = false and sp.rh_account_id = ?", acc).
		Joins("inner join package p on p.id = spkg.package_id").
		Joins("inner join package_name pn on pn.id = p.name_id").
		Where("spkg.rh_account_id = ?", acc).
		Where("pn.name = ?", pkgName)
}

// @Summary Show me all my systems which have a package installed
// @Description  Show me all my systems which have a package installed
// @ID packageSystems
// @Security RhIdentity
// @Accept   json
// @Produce  json
// @Param    package_name    path    string    true  "Package name"
// @Param    tags            query   []string  false "Tag filter"
// @Param    filter[system_profile][sap_system]   query  string  false "Filter only SAP systems"
// @Param    filter[system_profile][sap_sids][in] query []string  false "Filter systems by their SAP SIDs"
// @Success 200 {object} PackageSystemsResponse
// @Router /api/patch/v1/packages/{package_name}/systems [get]
func PackageSystemsListHandler(c *gin.Context) {
	account := c.GetInt(middlewares.KeyAccount)

	packageName := c.Param("package_name")
	if packageName == "" {
		c.JSON(http.StatusBadRequest, utils.ErrorResponse{Error: "package_name param not found"})
		return
	}

	query := packageSystemsQuery(account, packageName)
	query = ApplySearch(c, query, "sp.display_name")
	query, _ = ApplyTagsFilter(c, query, "sp.inventory_id")
	query, meta, links, err := ListCommon(query, c, fmt.Sprintf("/packages/%s/systems", packageName), PackageSystemsOpts)
	if err != nil {
		// Error handling and setting of result code & content is done in ListCommon
		return
	}

	var systems []PackageSystemItem
	err = query.Find(&systems).Error
	if err != nil {
		LogAndRespError(c, err, "database error")
		return
	}

	c.JSON(200, PackageSystemsResponse{
		Data:  systems,
		Links: *links,
		Meta:  *meta,
	})
}
