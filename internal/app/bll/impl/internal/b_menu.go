package internal

import (
	"context"

	"github.com/LyricTian/gin-admin/internal/app/errors"
	"github.com/LyricTian/gin-admin/internal/app/model"
	"github.com/LyricTian/gin-admin/internal/app/schema"
	"github.com/LyricTian/gin-admin/pkg/util"
)

// NewMenu 创建菜单管理实例
func NewMenu(trans model.ITrans,
	menu model.IMenu) *Menu {
	return &Menu{
		TransModel: trans,
		MenuModel:  menu,
	}
}

// Menu 菜单管理
type Menu struct {
	TransModel model.ITrans
	MenuModel  model.IMenu
}

// Query 查询数据
func (a *Menu) Query(ctx context.Context, params schema.MenuQueryParam, opts ...schema.MenuQueryOptions) (*schema.MenuQueryResult, error) {
	return a.MenuModel.Query(ctx, params, opts...)
}

// Get 查询指定数据
func (a *Menu) Get(ctx context.Context, recordID string, opts ...schema.MenuQueryOptions) (*schema.Menu, error) {
	item, err := a.MenuModel.Get(ctx, recordID, opts...)
	if err != nil {
		return nil, err
	} else if item == nil {
		return nil, errors.ErrNotFound
	}
	return item, nil
}

func (a *Menu) getSep() string {
	return "/"
}

// 获取父级路径
func (a *Menu) getParentPath(ctx context.Context, parentID string) (string, error) {
	if parentID == "" {
		return "", nil
	}

	pitem, err := a.MenuModel.Get(ctx, parentID)
	if err != nil {
		return "", err
	} else if pitem == nil {
		return "", errors.ErrMenuInvalidParent
	}

	var parentPath string
	if v := pitem.ParentPath; v != "" {
		parentPath = v + a.getSep()
	}
	parentPath += pitem.RecordID
	return parentPath, nil
}

func (a *Menu) getUpdate(ctx context.Context, recordID string) (*schema.Menu, error) {
	return a.Get(ctx, recordID, schema.MenuQueryOptions{
		IncludeActions:   true,
		IncludeResources: true,
	})
}

// Create 创建数据
func (a *Menu) Create(ctx context.Context, item schema.Menu) (*schema.Menu, error) {
	parentPath, err := a.getParentPath(ctx, item.ParentID)
	if err != nil {
		return nil, err
	}

	item.ParentPath = parentPath
	item.RecordID = util.MustUUID()
	err = a.MenuModel.Create(ctx, item)
	if err != nil {
		return nil, err
	}

	return a.getUpdate(ctx, item.RecordID)
}

// Update 更新数据
func (a *Menu) Update(ctx context.Context, recordID string, item schema.Menu) (*schema.Menu, error) {
	if recordID == item.ParentID {
		return nil, errors.ErrMenuNotAllowSelf
	}

	oldItem, err := a.MenuModel.Get(ctx, recordID)
	if err != nil {
		return nil, err
	} else if oldItem == nil {
		return nil, errors.ErrNotFound
	}
	item.ParentPath = oldItem.ParentPath

	err = ExecTrans(ctx, a.TransModel, func(ctx context.Context) error {
		// 如果父级更新，需要更新当前节点及节点下级的父级路径
		if item.ParentID != oldItem.ParentID {
			parentPath, err := a.getParentPath(ctx, item.ParentID)
			if err != nil {
				return err
			}
			item.ParentPath = parentPath

			opath := oldItem.ParentPath
			if opath != "" {
				opath += a.getSep()
			}
			opath += oldItem.RecordID

			result, err := a.MenuModel.Query(ctx, schema.MenuQueryParam{
				PrefixParentPath: opath,
			})
			if err != nil {
				return err
			}

			npath := item.ParentPath
			if npath != "" {
				npath += a.getSep()
			}
			npath += item.RecordID

			for _, menu := range result.Data {
				npath2 := npath + menu.ParentPath[len(opath):]
				err = a.MenuModel.UpdateParentPath(ctx, menu.RecordID, npath2)
				if err != nil {
					return err
				}
			}
		}

		return a.MenuModel.Update(ctx, recordID, item)
	})
	if err != nil {
		return nil, err
	}
	return a.getUpdate(ctx, recordID)
}

// Delete 删除数据
func (a *Menu) Delete(ctx context.Context, recordID string) error {
	result, err := a.MenuModel.Query(ctx, schema.MenuQueryParam{
		ParentID: &recordID,
	}, schema.MenuQueryOptions{PageParam: &schema.PaginationParam{PageSize: -1}})
	if err != nil {
		return err
	} else if result.PageResult.Total > 0 {
		return errors.ErrMenuNotAllowDelete
	}

	return a.MenuModel.Delete(ctx, recordID)
}
