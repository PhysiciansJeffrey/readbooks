
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
            name:'Home',
            component:Home
        },
        {
            path: '/resume/:id',
            name:'Resume',
            component: Resume
        },{
            path: '/detail/:id',
            name:'Detail',
            component: Detail
        },
        {
            path: '/search/:key',
            name:'Search',
            component: Search
        },
        {
            path: '/setting',
            name:'Setting',
            component: Setting
        }
    ]
})

export default router
