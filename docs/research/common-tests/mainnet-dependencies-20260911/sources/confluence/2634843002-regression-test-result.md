---
id: 2634843002
title: "Regression Test Result"
url: https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2634843002
version: 1
createdAt: 2026-04-20T02:11:00.862Z
fetched_at: 2026-09-11T07:30:00Z
---

## 테스트 케이스 

-  <https://wemade.atlassian.net/wiki/x/GQD2mg> 


## 테스트 환경

- 폐쇄망 서버

| IP | 역할 | 타입 |
| --- | --- | --- |
| (내부망 주소 7대, 보존본에서 생략) | Validator Node | 블록 생성·합의 |
| (내부망 주소 3대, 보존본에서 생략) | EN Node (Snap Sync) | 동기화 노드 |
| (내부망 주소 4대, 보존본에서 생략) | EN Node (Full Sync) | 동기화 노드 |
| (내부망 주소, 보존본에서 생략) | PN (EN + Bootnode) | P2P 연결 기준점 |


## 테스트 사전 작업

- gstable build
    - go v1.23.12
    - branch : dev
    - 기준 커밋 : <https://github.com/stable-net/go-stablenet/commit/e5a6e9e14c1e1d225c341b55798f31cd07b0bfcd> 
        - 추가 수정사항 : 테스트 환경 구성을 위해 기준 커밋에 별도 수정 적용
- pn (bootnode) 정보
    - (enode 주소와 내부망 IP는 보존본에서 생략)

(원문은 ac:structured-macro를 포함하며 markdown 변환에서 일부가 빠졌다. lossyConversion=true)
