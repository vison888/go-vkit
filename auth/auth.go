package auth

import (
	"path"

	"github.com/visonlv/go-vkit/logger"
)

type AuthRole struct {
	urls []string
	code string
}

type Auth struct {
	whiteUrls []string
	roles     map[string]*AuthRole
}

func NewAuth(whiteUrls []string) *Auth {
	return &Auth{
		whiteUrls: whiteUrls,
		roles:     make(map[string]*AuthRole),
	}
}

func (a *Auth) SetRole(r *AuthRole) {
	a.roles[r.code] = r
	logger.Infof("[auth] SetRole code:%s", r.code)
}

// TODO 性能可以优化 使用多层索引
func (a *Auth) IsPemission(roleList []string, url string) bool {
	//判断白名单
	for _, v := range a.whiteUrls {
		b, err := path.Match(v, url)
		if err != nil {
			logger.Errorf("[auth] IsPemission match url:%s err:%s", url, err)
			return false
		}
		if b {
			return true
		}
	}
	//判断各个角色权限 匹配一个则满足
	for _, code := range roleList {
		r, ok := a.roles[code]
		if ok {
			for _, v := range r.urls {
				b, err := path.Match(v, url)
				if err != nil {
					logger.Errorf("[auth] IsPemission match url:%s err:%s", url, err)
					return false
				}
				if b {
					return true
				}
			}
		}
	}
	return false
}
