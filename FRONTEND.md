# SmartTicket 前端技术文档

## 项目概述

SmartTicket 前端是面向企业自主部署的多租户工单与知识协作平台的用户界面，为不同角色用户提供现代化、响应式的 Web 应用体验。前端采用现代化技术栈，支持多语言、主题切换，并具备完善的动画效果和交互体验。

## 核心特色

### 🎨 **现代化用户体验**
- **响应式设计**: 支持桌面、平板、移动端
- **流畅动画**: GSAP + React Motion 提供专业级动画效果
- **主题切换**: 支持亮色/暗色主题切换
- **多语言支持**: 国际化 (i18n) 支持

### 🏢 **企业级功能**
- **多租户支持**: 租户级别的界面隔离和定制
- **角色化界面**: 根据用户角色展示不同功能模块
- **实时更新**: WebSocket 实现实时数据同步
- **离线支持**: PWA 特性支持基础离线功能

### 🚀 **高性能体验**
- **快速加载**: Vite 提供极速开发服务器和构建
- **代码分割**: 路由级别的代码分割
- **状态管理**: Redux Toolkit 高效状态管理
- **缓存策略**: 智能数据缓存和预加载

## 技术栈

### 核心框架
- **构建工具**: Vite 4.x
- **语言**: TypeScript 5.x
- **框架**: React 18.x
- **状态管理**: Redux Toolkit + RTK Query
- **路由**: React Router v6

### UI 与样式
- **UI 库**: Ant Design 5.x
- **CSS 框架**: Tailwind CSS 3.x
- **样式方案**: CSS Modules + Tailwind
- **图标**: Ant Design Icons + Lucide React

### 动画与交互
- **动画库**: GSAP (GreenSock) + React Motion
- **手势**: React Use Gesture
- **拖拽**: React DnD
- **虚拟滚动**: React Window

### 工具与库
- **HTTP 客户端**: Axios
- **表单处理**: React Hook Form + Zod
- **日期处理**: Day.js
- **工具库**: Lodash-es
- **类型检查**: TypeScript + ESLint + Prettier

### 测试
- **测试框架**: Vitest + React Testing Library
- **E2E 测试**: Playwright
- **覆盖率要求**: ≥ 70%
- **视觉回归**: Chromatic (可选)

## 项目架构

### 目录结构
```
smartticket-frontend/
├── public/                     # 静态资源
│   ├── locales/               # 多语言文件
│   ├── icons/                 # 图标文件
│   └── index.html
├── src/
│   ├── components/            # 可复用组件
│   │   ├── common/           # 通用组件
│   │   ├── forms/            # 表单组件
│   │   ├── charts/           # 图表组件
│   │   └── ui/               # UI 组件
│   ├── pages/                 # 页面组件
│   │   ├── auth/             # 认证页面
│   │   ├── tickets/          # 工单页面
│   │   ├── knowledge/        # 知识库页面
│   │   ├── admin/            # 管理页面
│   │   └── dashboard/        # 仪表板
│   ├── features/              # 功能模块
│   │   ├── auth/             # 认证功能
│   │   ├── tickets/          # 工单功能
│   │   ├── knowledge/        # 知识库功能
│   │   ├── notifications/     # 通知功能
│   │   └── theme/            # 主题功能
│   ├── store/                 # Redux 状态管理
│   │   ├── slices/           # Redux slices
│   │   ├── api/              # RTK Query API
│   │   └── middleware/       # Redux 中间件
│   ├── hooks/                 # 自定义 Hooks
│   │   ├── useAuth.ts        # 认证 Hook
│   │   ├── useTheme.ts       # 主题 Hook
│   │   ├── useLocalStorage.ts # 本地存储 Hook
│   │   └── useWebSocket.ts   # WebSocket Hook
│   ├── services/              # 服务层
│   │   ├── api.ts            # API 客户端
│   │   ├── auth.ts           # 认证服务
│   │   ├── storage.ts        # 存储服务
│   │   └── websocket.ts      # WebSocket 服务
│   ├── utils/                 # 工具函数
│   │   ├── constants.ts      # 常量定义
│   │   ├── helpers.ts        # 辅助函数
│   │   ├── validators.ts     # 验证函数
│   │   └── formatters.ts     # 格式化函数
│   ├── styles/                # 样式文件
│   │   ├── globals.css       # 全局样式
│   │   ├── components.css    # 组件样式
│   │   └── animations.css    # 动画样式
│   ├── assets/                # 静态资源
│   │   ├── images/           # 图片
│   │   ├── icons/            # 图标
│   │   └── fonts/            # 字体
│   ├── types/                 # TypeScript 类型定义
│   │   ├── api.ts            # API 类型
│   │   ├── auth.ts           # 认证类型
│   │   ├── ticket.ts         # 工单类型
│   │   └── common.ts         # 通用类型
│   ├── i18n/                  # 国际化
│   │   ├── locales/          # 语言文件
│   │   └── index.ts          # i18n 配置
│   ├── theme/                 # 主题配置
│   │   ├── light.ts          # 亮色主题
│   │   ├── dark.ts           # 暗色主题
│   │   └── index.ts          # 主题管理
│   ├── App.tsx                # 根组件
│   ├── main.tsx               # 应用入口
│   └── vite-env.d.ts          # Vite 类型声明
├── tests/                     # 测试文件
│   ├── __mocks__/             # Mock 文件
│   ├── setup.ts              # 测试配置
│   ├── components/           # 组件测试
│   ├── pages/                # 页面测试
│   └── e2e/                  # E2E 测试
├── docs/                      # 文档
├── scripts/                   # 构建脚本
├── package.json               # 依赖配置
├── vite.config.ts             # Vite 配置
├── tsconfig.json              # TypeScript 配置
├── tailwind.config.js         # Tailwind 配置
├── .eslintrc.js               # ESLint 配置
├── .prettierrc                # Prettier 配置
├── vitest.config.ts           # Vitest 配置
└── playwright.config.ts       # Playwright 配置
```

### 架构设计原则

#### 1. 组件化设计
- **原子设计**: 原子 → 分子 → 有机体 → 模板 → 页面
- **可复用性**: 高度可复用的 UI 组件库
- **一致性**: 统一的设计系统和交互规范

#### 2. 状态管理架构
```
┌─────────────────────────────────────────────────────────────┐
│                        Redux Store                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│  │    Slices        │ │     API          │ │   Middleware    │  │
│  │                │ │                │ │                │  │
│  │ • authSlice     │ │ • ticketsApi     │ │ • authMiddleware │  │
│  │ • ticketsSlice  │ │ • knowledgeApi   │ │ • loggerMiddleware│ │
│  │ • themeSlice    │ │ • usersApi       │ │ • persistMiddleware│ │
│  │ • uiSlice       │ │ • llmApi         │ │ • rtkQueryCache   │  │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

#### 3. 组件层次结构
```
App
├── Layout
│   ├── Header
│   │   ├── Logo
│   │   ├── Navigation
│   │   ├── UserMenu
│   │   └── ThemeToggle
│   ├── Sidebar
│   │   ├── MenuItems
│   │   └── CollapsibleSection
│   └── Footer
├── Routes
│   ├── AuthPages
│   │   ├── Login
│   │   └── Register
│   ├── Dashboard
│   ├── TicketPages
│   │   ├── TicketList
│   │   ├── TicketDetail
│   │   └── CreateTicket
│   ├── KnowledgePages
│   │   ├── ArticleList
│   │   ├── ArticleDetail
│   │   └── CreateArticle
│   └── AdminPages
│       ├── UserManagement
│       ├── TenantSettings
│       └── SystemConfiguration
└── CommonComponents
    ├── Modals
    ├── Notifications
    ├── LoadingStates
    └── ErrorBoundaries
```

## 核心功能模块

### 1. 认证模块 (Auth)
```typescript
// 状态管理
interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  loading: boolean;
  error: string | null;
}

// 核心功能
- JWT Token 管理
- 多租户登录
- 角色权限验证
- 会话持久化
- 自动登出
```

### 2. 工单管理 (Tickets)
```typescript
// 功能特性
- 工单列表与筛选
- 工单创建与编辑
- 状态流转可视化
- SLA 计时器
- 实时通知
- 批量操作
- 附件上传
- 评论系统
```

### 3. 知识库 (Knowledge)
```typescript
// 功能特性
- 文章浏览与搜索
- 分类管理
- 版本控制
- 发布流程
- 标签系统
- 收藏功能
- 相关推荐
```

### 4. 仪表板 (Dashboard)
```typescript
// 功能特性
- 数据可视化图表
- 工单统计
- SLA 合规率
- 性能指标
- 实时数据更新
- 自定义面板
```

### 5. 管理后台 (Admin)
```typescript
// 功能特性
- 用户管理
- 租户配置
- 系统设置
- 审计日志
- 数据导入导出
- LLM 配置
```

## 动画与交互设计

### GSAP 动画体系
```typescript
// 页面转场动画
const pageTransition = {
  initial: { opacity: 0, y: 20 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -20 }
};

// 微交互动画
const microInteractions = {
  hover: { scale: 1.05, transition: { duration: 0.2 } },
  click: { scale: 0.95, transition: { duration: 0.1 } },
  loading: { rotate: 360, repeat: Infinity }
};

// 复杂动画场景
const complexAnimations = {
  dashboard: {
    chartEntrance: "fadeUp 0.6s ease-out",
    numberCounter: "countUp 1s ease-out",
    cardStagger: "stagger 0.1s ease-out"
  },
  ticketFlow: {
    statusChange: "statusPulse 0.3s ease-out",
    notification: "slideIn 0.4s ease-out",
    priorityUpdate: "flash 0.5s ease-out"
  }
};
```

### React Motion 交互
```typescript
// 手势支持
const gestures = {
  swipe: "支持左右滑动手势",
  pinch: "支持缩放手势",
  drag: "支持拖拽操作"
};

// 物理动画
const physics = {
  spring: "弹性动画效果",
  inertia: "惯性滚动效果",
  bounce: "弹跳动画效果"
};
```

## 多语言支持

### 国际化架构
```typescript
// 支持语言
const supportedLanguages = {
  'en-US': 'English',
  'zh-CN': '简体中文',
  'zh-TW': '繁體中文',
  'ja-JP': '日本語',
  'ko-KR': '한국어',
  'de-DE': 'Deutsch',
  'fr-FR': 'Français'
};

// 语言包结构
interface LanguagePackage {
  common: {
    buttons: Record<string, string>;
    messages: Record<string, string>;
    errors: Record<string, string>;
  };
  auth: {
    login: Record<string, string>;
    register: Record<string, string>;
  };
  tickets: {
    status: Record<string, string>;
    priority: Record<string, string>;
    actions: Record<string, string>;
  };
  // ... 其他模块
}
```

## 主题系统

### 主题配置
```typescript
// 基础主题配置
interface ThemeConfig {
  colors: {
    primary: string;
    secondary: string;
    success: string;
    warning: string;
    error: string;
    background: string;
    surface: string;
    text: {
      primary: string;
      secondary: string;
      disabled: string;
    };
  };
  spacing: Record<string, string>;
  typography: {
    fontFamily: string;
    fontSize: Record<string, string>;
    lineHeight: Record<string, string>;
  };
  borderRadius: Record<string, string>;
  shadows: Record<string, string>;
  animations: {
    duration: Record<string, string>;
    easing: Record<string, string>;
  };
}

// 主题切换实现
const themeManager = {
  light: lightTheme,
  dark: darkTheme,
  auto: 'system-preference',
  custom: 'user-defined-theme'
};
```

## 性能优化策略

### 代码分割
```typescript
// 路由级别分割
const routes = [
  {
    path: '/',
    component: lazy(() => import('./pages/Dashboard'))
  },
  {
    path: '/tickets',
    component: lazy(() => import('./pages/tickets/TicketList'))
  },
  // ... 其他路由
];

// 组件级别分割
const HeavyChart = lazy(() => import('./components/HeavyChart'));
const AdminPanel = lazy(() => import('./components/AdminPanel'));
```

### 缓存策略
```typescript
// RTK Query 缓存配置
const api = createApi({
  baseQuery: fetchBaseQuery({
    baseUrl: '/api/v1',
    prepareHeaders: (headers, { getState }) => {
      const token = (getState() as RootState).auth.token;
      if (token) {
        headers.set('authorization', `Bearer ${token}`);
      }
      return headers;
    },
  }),
  tagTypes: ['Ticket', 'User', 'KnowledgeArticle', 'LLMProvider'],
  endpoints: (builder) => ({
    // API 定义
  })
});
```

### 预加载与预获取
```typescript
// 预加载关键数据
const preloadCriticalData = {
  userPermissions: '/api/v1/auth/permissions',
  tenantSettings: '/api/v1/tenant/settings',
  userNotifications: '/api/v1/notifications/unread'
};

// 智能预获取
const prefetchStrategies = {
  hover: '鼠标悬停预获取',
  idle: '空闲时间预获取',
  network: '网络良好时预获取'
};
```

## 测试策略

### 测试覆盖率要求
- **单元测试**: ≥ 70%
- **组件测试**: 覆盖所有可复用组件
- **集成测试**: 覆盖关键用户流程
- **E2E 测试**: 覆盖核心业务场景

### 测试工具配置
```typescript
// Vitest 配置
export default defineConfig({
  testEnvironment: 'jsdom',
  setupFiles: ['./tests/setup.ts'],
  coverage: {
    provider: 'v8',
    reporter: ['text', 'html', 'lcov'],
    thresholds: {
      global: {
        branches: 70,
        functions: 70,
        lines: 70,
        statements: 70
      }
    }
  }
});

// React Testing Library 配置
const renderWithProviders = (
  ui: React.ReactElement,
  options: RenderOptions = {}
) => {
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <Provider store={store}>
      <BrowserRouter>
        <I18nextProvider i18n={i18n}>
          <ThemeProvider theme={theme}>
            {children}
          </ThemeProvider>
        </I18nextProvider>
      </BrowserRouter>
    </Provider>
  );

  return render(ui, { wrapper: Wrapper, ...options });
};
```

### 测试结构
```
tests/
├── __mocks__/                 # Mock 文件
│   ├── api.ts               # API Mock
│   ├── websocket.ts         # WebSocket Mock
│   └── localStorage.ts      # LocalStorage Mock
├── setup.ts                  # 测试配置
├── utils/                    # 测试工具
│   ├── renderWithProviders.ts
│   ├── testUtils.ts
│   └── mockData.ts
├── components/               # 组件测试
│   ├── common/
│   ├── forms/
│   └── ui/
├── pages/                    # 页面测试
│   ├── auth/
│   ├── tickets/
│   └── dashboard/
├── features/                 # 功能测试
│   ├── auth/
│   ├── tickets/
│   └── theme/
├── hooks/                    # Hook 测试
└── e2e/                      # E2E 测试
    ├── auth.spec.ts
    ├── tickets.spec.ts
    └── dashboard.spec.ts
```

## 开发环境配置

### 环境要求
```bash
# Node.js 版本
node --version  # >= 18.0.0

# 包管理器
npm --version    # >= 8.0.0
# 或
pnpm --version   # >= 7.0.0
```

### 快速开始
```bash
# 1. 安装依赖
npm install

# 2. 启动开发服务器
npm run dev

# 3. 运行测试
npm run test

# 4. 构建生产版本
npm run build

# 5. 预览生产版本
npm run preview
```

### 开发脚本
```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "test": "vitest",
    "test:ui": "vitest --ui",
    "test:coverage": "vitest --coverage",
    "lint": "eslint . --ext ts,tsx --report-unused-disable-directives --max-warnings 0",
    "lint:fix": "eslint . --ext ts,tsx --fix",
    "format": "prettier --write .",
    "type-check": "tsc --noEmit",
    "e2e": "playwright test",
    "e2e:ui": "playwright test --ui"
  }
}
```

## 部署配置

### 构建优化
```typescript
// Vite 生产配置
export default defineConfig({
  build: {
    outDir: 'dist',
    sourcemap: false,
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true
      }
    },
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          antd: ['antd'],
          charts: ['recharts', 'echarts']
        }
      }
    },
    chunkSizeWarningLimit: 1000
  }
});
```

### 环境变量配置
```typescript
// 环境变量类型定义
interface ImportMetaEnv {
  readonly VITE_API_URL: string;
  readonly VITE_WS_URL: string;
  readonly VITE_APP_TITLE: string;
  readonly VITE_APP_VERSION: string;
  readonly VITE_ENABLE_MOCK: string;
  readonly VITE_SENTRY_DSN: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
```

## 监控与错误处理

### 错误边界
```typescript
// 全局错误边界
class ErrorBoundary extends Component<
  PropsWithChildren,
  { hasError: boolean; error?: Error }
> {
  constructor(props: PropsWithChildren) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
    // 发送错误到监控服务
    reportError(error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <Result
          status="500"
          title="Something went wrong"
          subTitle="Please try refreshing the page or contact support"
          extra={<Button onClick={() => window.location.reload()}>Refresh</Button>}
        />
      );
    }

    return this.props.children;
  }
}
```

### 性能监控
```typescript
// 性能指标监控
const performanceMetrics = {
  // Core Web Vitals
  FCP: 'First Contentful Paint',
  LCP: 'Largest Contentful Paint',
  FID: 'First Input Delay',
  CLS: 'Cumulative Layout Shift',

  // 自定义指标
  routeChangeTime: '路由切换时间',
  apiResponseTime: 'API 响应时间',
  componentRenderTime: '组件渲染时间',

  // 业务指标
  ticketCreationTime: '工单创建时间',
  searchResponseTime: '搜索响应时间',
  pageLoadTime: '页面加载时间'
};
```

## 总结

SmartTicket 前端采用现代化技术栈，具备企业级应用的所有特性：

### 🚀 **技术优势**
- **现代技术栈**: Vite + React + TypeScript + Redux Toolkit
- **企业级 UI**: Ant Design + Tailwind CSS
- **专业动画**: GSAP + React Motion
- **完善测试**: 70%+ 测试覆盖率
- **国际化**: 多语言支持
- **主题化**: 亮色/暗色主题切换

### 🎯 **用户体验**
- **响应式设计**: 适配所有设备
- **流畅交互**: 专业级动画效果
- **实时更新**: WebSocket 实时数据同步
- **离线支持**: PWA 特性

### 🔧 **开发体验**
- **快速开发**: Vite 极速热更新
- **类型安全**: TypeScript 全栈类型支持
- **代码质量**: ESLint + Prettier 自动化
- **调试友好**: 完善的开发工具集成

通过这套技术栈，SmartTicket 前端能够提供现代化、高性能、易维护的企业级用户界面。