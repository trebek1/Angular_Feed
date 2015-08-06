package mock

import (
	"errors"
	"fmt"
	server "qbase/synthos/synthos_svr"
)

type MockEntityAnnotator struct {
	SimulateNoInfo bool
}

func (db *MockEntityAnnotator) FetchEntityInfo(entityType server.EntityType, entityId int) (info server.StringProps, err error) {
	isValidEntityId := (entityId >= 0)
	if !isValidEntityId {
		return server.StringProps{}, errors.New(fmt.Sprintf("Invalid entityId: %v", entityId))
	}

	if db.SimulateNoInfo {
		return nil, nil
	} else {
		gibberish := "Orem ipsum dolor sit amet, consectetur adipiscing elit, sed do " +
			" eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad " +
			" minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip " +
			" ex ea commodo consequat."

		genericPersonImageUrl := "http://cdn0.vox-cdn.com/images/verge/default-avatar.v9899025.gif"

		return server.StringProps{
			"label":       fmt.Sprintf("%v:%v", entityType, entityId),
			"description": gibberish,
			"imageUrl":    genericPersonImageUrl,
		}, nil
	}
}
