<h1 align="center">cdms</h1>
<div align="center">
Continuous Deployment Management System-持续部署管理系统
<p align="center">
<img src="https://img.shields.io/badge/Golang-1.21-brightgreen" alt="Go version"/>
<img src="https://img.shields.io/badge/Temporal-1.21-brightgreen" alt="Temporal version"/>  
</p>
</div>

## 主要功能
- `流水线作业平台`(基于Temporal工作流引擎)--进行中
- -  git仓库拉取代码-基于[go-git](https://github.com/go-git/go-git)  
- -  容器镜像构建-基于[buildkit](https://github.com/moby/buildkit) 
- -  容器镜像构建-基于[kaniko](https://github.com/GoogleContainerTools/kaniko)
- `CMDB`(容器化部署专用CMDB)--计划中

## 框架类库
- [go-gin-rest-api](https://github.com/linclin/go-gin-rest-api) 后台基础框架
- [Temporal](https://github.com/temporalio/temporal) 工作流引擎
 