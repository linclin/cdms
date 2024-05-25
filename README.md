<h1 align="center">cdms</h1>
<div align="center">
Continuous Deployment Management System-持续部署管理系统
<p align="center">
<img src="https://img.shields.io/badge/Golang-1.22-brightgreen" alt="Go version">  
</p>
</div>

## 主要功能
- `流水线作业平台` 
- -  DAG流水线-基于[fastflow](https://github.com/linclin/fastflow)  
- `CI`(容器构建)
- -  git仓库拉取代码-基于[go-git](https://github.com/go-git/go-git)  
- -  容器镜像构建-Docker API基于[docker](github.com/docker/docker/api)
- -  容器镜像构建-k8s Job 基于[buildkit](https://github.com/moby/buildkit) 
- -  容器镜像构建-k8s Job 基于[kaniko](https://github.com/GoogleContainerTools/kaniko)
- -  制品包上传对象存储-[AWS s3](https://github.com/aws/aws-sdk-go-v2)
- -  镜像同步- 基于[google go-containerregistry](https://github.com/google/go-containerregistry)

- `CD`(容器部署)
- -  镜像预热- 基于[openkruise ImagePullJob](https://openkruise.io/zh/docs/user-manuals/imagepulljob/)  
- -  Helm部署-基于[Helm go SDK](https://github.com/helm/helm)

## 框架类库
- [go-gin-rest-api](https://github.com/linclin/go-gin-rest-api) 后台基础框架
- [fastflow](https://github.com/linclin/fastflow) DAG流水线，基于https://github.com/ShiningRush/fastflow开发
 