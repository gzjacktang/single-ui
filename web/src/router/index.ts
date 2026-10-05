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
      meta: { title: 'SBOX · 登录' },
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
          meta: { title: 'SBOX · 入站管理' },
        },
        {
          path: 'user',
          component: () => import('../view/panel/user/User.vue'),
          meta: { title: 'SBOX · 用户管理' },
        },
        {
          path: 'endpoint',
          component: () => import('../view/panel/endpoint/Endpoint.vue'),
          meta: { title: 'SBOX · 端点管理' },
        },
        {
          path: 'core',
          component: () => import('../view/panel/core/Core.vue'),
          meta: { title: 'SBOX · 核心配置' },
        },
        {
          path: 'config',
          component: () => import('../view/panel/config/Config.vue'),
          meta: { title: 'SBOX · 面板配置' },
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
  document.title = (to.meta.title as string) || 'SBOX'

  const token = localStorage.getItem('token')

  if (token && to.path === '/login') {
    return '/'
  }

  if (!token && to.path !== '/login') {
    return '/login'
  }
})

export default router
