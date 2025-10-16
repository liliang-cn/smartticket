-- Create knowledge base enums
CREATE TYPE knowledge_status AS ENUM ('Draft', 'Review', 'Published', 'Archived');
CREATE TYPE knowledge_visibility AS ENUM ('Public', 'Internal', 'Restricted');

-- Create knowledge categories
CREATE TABLE knowledge_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES knowledge_categories(id),
    icon VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for knowledge categories
CREATE INDEX idx_knowledge_categories_tenant_id ON knowledge_categories(tenant_id);
CREATE INDEX idx_knowledge_categories_parent_id ON knowledge_categories(parent_id);

-- Enable RLS for knowledge categories
ALTER TABLE knowledge_categories ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_knowledge_categories ON knowledge_categories
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Create knowledge articles table
CREATE TABLE knowledge_articles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    summary TEXT,
    category_id UUID REFERENCES knowledge_categories(id),
    author_id UUID NOT NULL REFERENCES users(id),
    status knowledge_status NOT NULL DEFAULT 'Draft',
    visibility knowledge_visibility NOT NULL DEFAULT 'Internal',
    language VARCHAR(10) NOT NULL DEFAULT 'en',
    tags TEXT[] DEFAULT '{}',
    view_count INTEGER NOT NULL DEFAULT 0,
    helpful_count INTEGER NOT NULL DEFAULT 0,
    not_helpful_count INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    published_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for knowledge articles
CREATE INDEX idx_knowledge_articles_tenant_id ON knowledge_articles(tenant_id);
CREATE INDEX idx_knowledge_articles_status ON knowledge_articles(status);
CREATE INDEX idx_knowledge_articles_visibility ON knowledge_articles(visibility);
CREATE INDEX idx_knowledge_articles_category_id ON knowledge_articles(category_id);
CREATE INDEX idx_knowledge_articles_author_id ON knowledge_articles(author_id);
CREATE INDEX idx_knowledge_articles_published_at ON knowledge_articles(published_at);
CREATE INDEX idx_knowledge_articles_language ON knowledge_articles(language);
CREATE INDEX idx_knowledge_articles_deleted ON knowledge_articles(is_deleted);
CREATE INDEX idx_knowledge_articles_tags ON knowledge_articles USING GIN(tags);

-- Full-text search index
CREATE INDEX idx_knowledge_articles_search ON knowledge_articles USING GIN(
    to_tsvector('english', title || ' ' || COALESCE(summary, '') || ' ' || COALESCE(content, ''))
);

-- Create trigger for updated_at
CREATE TRIGGER update_knowledge_articles_updated_at
    BEFORE UPDATE ON knowledge_articles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Enable RLS
ALTER TABLE knowledge_articles ENABLE ROW LEVEL SECURITY;

-- RLS policies for knowledge articles
CREATE POLICY tenant_isolation_knowledge_articles ON knowledge_articles
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid AND NOT is_deleted);

-- Published articles are accessible to all users in tenant
CREATE POLICY published_articles_visible ON knowledge_articles
    FOR SELECT
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        AND status = 'Published'
        AND (expires_at IS NULL OR expires_at > NOW())
    );

-- Internal articles visible to support staff
CREATE POLICY internal_articles_visible ON knowledge_articles
    FOR SELECT
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        AND visibility IN ('Internal', 'Public')
        AND EXISTS (
            SELECT 1 FROM users
            WHERE id = current_setting('app.current_user_id', true)::uuid
            AND tenant_id = knowledge_articles.tenant_id
            AND role IN ('admin', 'team_lead')
        )
    );

-- Draft articles visible to author and admins
CREATE POLICY draft_articles_visible ON knowledge_articles
    FOR SELECT
    USING (
        tenant_id = current_setting('app.current_tenant_id', true)::uuid
        AND (
            author_id = current_setting('app.current_user_id', true)::uuid
            OR EXISTS (
                SELECT 1 FROM users
                WHERE id = current_setting('app.current_user_id', true)::uuid
                AND tenant_id = knowledge_articles.tenant_id
                AND role IN ('admin', 'team_lead')
            )
        )
    );

-- Create article view tracking table
CREATE TABLE knowledge_article_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    article_id UUID NOT NULL REFERENCES knowledge_articles(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    ip_address INET,
    user_agent TEXT,
    viewed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for article views
CREATE INDEX idx_knowledge_article_views_article_id ON knowledge_article_views(article_id);
CREATE INDEX idx_knowledge_article_views_user_id ON knowledge_article_views(user_id);
CREATE INDEX idx_knowledge_article_views_tenant_id ON knowledge_article_views(tenant_id);
CREATE INDEX idx_knowledge_article_views_viewed_at ON knowledge_article_views(viewed_at);

-- Enable RLS for article views
ALTER TABLE knowledge_article_views ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_article_views ON knowledge_article_views
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);