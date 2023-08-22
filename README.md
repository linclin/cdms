<h1 align="center">cdms</h1>
<div align="center">
(Continuous Deployment Management System)-持续部署管理系统
<p align="center">
<img src="https://img.shields.io/badge/Golang-1.20.3-brightgreen" alt="Go version"/>
<img src="https://img.shields.io/badge/Gin-1.9.0-brightgreen" alt="Gin version"/>
<img src="https://img.shields.io/badge/Gorm-1.24.6-brightgreen" alt="Gorm version"/> 
</p>
</div>

## 主要功能
- 流水线作业平台(基于DAG设计的流水线作业调度平台)--进行中
- CMDB(面向容器化的系统和子系统管理)
- 容器部署API(基于K8S官方client-go封装实现部署、重启、扩容、回滚能力)

## 主要框架类库
- [go-gin-rest-api](https://github.com/linclin/go-gin-rest-api) 后台基础框架
- [fastflow](https://github.com/linclin/fastflow) DAG流水线作业库，基于[fastflow](https://github.com/ShiningRush/fastflow)二次开发改造
 