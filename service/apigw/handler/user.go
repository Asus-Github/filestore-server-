package handler

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	cmn "filestore-server/common"
	cfg "filestore-server/config"
	"filestore-server/middleware"
	userProto "filestore-server/service/account/proto"
)

// Handler : 网关接口处理器
type Handler struct {
	UserCli userProto.UserService
}

// rpcFailed : rpc调用失败时统一返回 503
func rpcFailed(c *gin.Context, err error) {
	log.Println("rpc call failed:", err)
	middleware.Resp(c, http.StatusServiceUnavailable, cmn.StatusServerError, "服务暂不可用, 请稍后重试", nil)
}

// SignupHandler : 响应注册页面
func SignupHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signup.html")
}

// DoSignupHandler : 处理注册post请求
func (h *Handler) DoSignupHandler(c *gin.Context) {
	resp, err := h.UserCli.Signup(c.Request.Context(), &userProto.ReqSignup{
		Username: strings.TrimSpace(c.PostForm("username")),
		Password: c.PostForm("password"),
	})
	if err != nil {
		rpcFailed(c, err)
		return
	}
	middleware.Resp(c, http.StatusOK, resp.Code, resp.Message, nil)
}

// SigninHandler : 响应登录页面
func SigninHandler(c *gin.Context) {
	c.Redirect(http.StatusFound, "/static/view/signin.html")
}

// entries : 上传/下载服务的入口地址
func entries(c *gin.Context) (string, string) {
	return middleware.ServiceEntry(c, cfg.UploadLBHost, cfg.UploadServiceHost),
		middleware.ServiceEntry(c, cfg.DownloadLBHost, cfg.DownloadServiceHost)
}

// DoSigninHandler : 处理登录post请求
func (h *Handler) DoSigninHandler(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	rpcResp, err := h.UserCli.Signin(c.Request.Context(), &userProto.ReqSignin{
		Username: username,
		Password: c.PostForm("password"),
	})
	if err != nil {
		rpcFailed(c, err)
		return
	}
	if rpcResp.Code != cmn.StatusOK {
		middleware.Fail(c, rpcResp.Code, rpcResp.Message)
		return
	}

	// 登录成功，返回用户信息及上传/下载入口
	if rpcResp.Username != "" {
		username = rpcResp.Username
	}
	upEntry, dlEntry := entries(c)
	middleware.OK(c, "登录成功", gin.H{
		"Location":      "/static/view/home.html",
		"Username":      username,
		"Token":         rpcResp.Token,
		"ExpireAt":      rpcResp.ExpireAt,
		"UploadEntry":   upEntry,
		"DownloadEntry": dlEntry,
	})
}

// SignoutHandler : 退出登录, 注销当前token
func (h *Handler) SignoutHandler(c *gin.Context) {
	resp, err := h.UserCli.Signout(c.Request.Context(), &userProto.ReqSignout{
		Username: middleware.Username(c),
		Token:    middleware.ExtractToken(c),
	})
	if err != nil {
		rpcFailed(c, err)
		return
	}
	middleware.Resp(c, http.StatusOK, resp.Code, resp.Message, nil)
}

// UserInfoHandler ： 查询当前登录用户的信息
func (h *Handler) UserInfoHandler(c *gin.Context) {
	resp, err := h.UserCli.UserInfo(c.Request.Context(), &userProto.ReqUserInfo{
		Username: middleware.Username(c),
	})
	if err != nil {
		rpcFailed(c, err)
		return
	}
	if resp.Code != cmn.StatusOK {
		middleware.Fail(c, resp.Code, resp.Message)
		return
	}

	upEntry, dlEntry := entries(c)
	middleware.OK(c, "OK", gin.H{
		"Username":      resp.Username,
		"Email":         resp.Email,
		"Phone":         resp.Phone,
		"SignupAt":      resp.SignupAt,
		"LastActive":    resp.LastActiveAt,
		"Status":        resp.Status,
		"UploadEntry":   upEntry,
		"DownloadEntry": dlEntry,
		"ServerTime":    time.Now().Unix(),
	})
}
