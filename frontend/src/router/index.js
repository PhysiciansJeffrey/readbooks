
import { createRouter, createWebHashHistory } from 'vue-router'


const Home = () => import('@/views/Home.vue')
const Resume = ()=>import('@/views/Resume.vue')
const Detail = ()=>import('@/views/Detail.vue')
const Search = ()=>import('@/views/Search.vue')
// const JM = ()=>import('@/views/JM.vue')
const Setting = ()=>import('@/views/Setting.vue')

const router=createRouter({
    history:createWebHashHistory(),
    routes:[
        {
            path:'/',
            component:Home
        },
        {
            path: '/resume/:id',
            component: Resume
        },{
            path: '/detail/:id',
            component: Detail
        },
        {
            path: '/search/:key',
            component: Search
        },
        {
            path: '/setting',
            component: Setting
        }
    ]
})

export default router
