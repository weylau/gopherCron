package etcd_func

import (
	"github.com/gin-gonic/gin"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"ojbk.io/gopherCron/cmd/service/request"
	"ojbk.io/gopherCron/common"
	"ojbk.io/gopherCron/errors"
	"ojbk.io/gopherCron/pkg/db"
	"ojbk.io/gopherCron/pkg/etcd"
	"ojbk.io/gopherCron/utils"
)

type KillTaskRequest struct {
	ProjectID string `form:"project_id" binding:"required"`
	TaskID    string `form:"task_id" binding:"required"`
}

// KillTask kill task from etcd
// post project & task name
// 一般强行结束任务就是想要停止任务 所以这里对任务的状态进行变更
func KillTask(c *gin.Context) {
	var (
		err       error
		req       KillTaskRequest
		task      *common.TaskInfo
		errObj    errors.Error
		uid       string
		userID    primitive.ObjectID
		projectID primitive.ObjectID
	)

	if err = utils.BindArgsWithGin(c, &req); err != nil {
		errObj = errors.ErrInvalidArgument
		errObj.Log = "[Controller - KillTask] KillTaskRequest args error:" + err.Error()
		request.APIError(c, errObj)
		return
	}

	uid = c.GetString("jwt_user")

	if userID, err = primitive.ObjectIDFromHex(uid); err != nil {
		request.APIError(c, errors.ErrInvalidArgument)
		return
	}

	if projectID, err = primitive.ObjectIDFromHex(req.ProjectID); err != nil {
		request.APIError(c, errors.ErrInvalidArgument)
		return
	}

	// 这里主要是防止用户强杀不属于自己项目的业务
	if _, err = db.CheckProjectExist(projectID, userID); err != nil {
		request.APIError(c, err)
		return
	}

	if err = etcd.Manager.KillTask(req.ProjectID, req.TaskID); err != nil {
		request.APIError(c, err)
		return
	}

	// 强杀任务后暂停任务
	if task, err = etcd.Manager.GetTask(req.ProjectID, req.TaskID); err != nil {
		goto ChangeStatusError
	}

	task.Status = 0

	if _, err = etcd.Manager.SaveTask(task); err != nil {
		goto ChangeStatusError
	}
	request.APISuccess(c, nil)
	return

ChangeStatusError:
	errObj = err.(errors.Error)
	errObj.Msg = "强行关闭任务成功，暂停任务失败"
	errObj.MsgEn = "task killed, not stop"
	request.APIError(c, errObj)
}
