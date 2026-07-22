import { createRouter, createWebHistory } from 'vue-router'
import Auth from '../view/auth/Login.vue'
import Panel from '../view/panel/Panel.vue'

const panelPath = (window.__PANEL_PATH__ || '/').replace(/\/?$/, '/')

const router = createRouter({
  history: createWebHistory(panelPath),
  routes: [
    {
      path: '/login',
      component: Auth,
      meta: { title: 'SLINX · 登录' },
    },
    {
      path: '/',
      component: Panel,
      children: [
        {
          path: '',
          redirect: '/inbound',
        },
        {
          path: 'inbound',
          component: () => import('../view/panel/inbound/Inbound.vue'),
          meta: { title: 'SLINX · 入站管理' },
        },
        {
          path: 'user',
          component: () => import('../view/panel/user/User.vue'),
          meta: { title: 'SLINX · 用户管理' },
        },
        {
          path: 'endpoint',
          component: () => import('../view/panel/endpoint/Endpoint.vue'),
          meta: { title: 'SLINX · 端点管理' },
        },
        {
          path: 'detect',
          component: () => import('../view/panel/detect/Detect.vue'),
          meta: { title: 'SLINX · IP检测' },
        },
        {
          path: 'core',
          component: () => import('../view/panel/core/Core.vue'),
          meta: { title: 'SLINX · 核心配置' },
        },
        {
          path: 'config',
          component: () => import('../view/panel/config/Config.vue'),
          meta: { title: 'SLINX · 面板配置' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/login',
    },
  ],
})

router.beforeEach((to) => {
  document.title = (to.meta.title as string) || 'SLINX'

  const token = localStorage.getItem('token')

  if (token && to.path === '/login') {
    return '/'
  }

  if (!token && to.path !== '/login') {
    return '/login'
  }
})

export default router
