CREATE TABLE snippet (
  id uuid primary key,
  creator_ip inet not null,
  content text not null,
  created_at timestamptz not null
);

CREATE INDEX ON snippet(created_at, creator_ip);

CREATE TABLE views (
  snippet_id uuid not null references snippet(id),
  viewer_ip inet,
  viewed_at timestamptz not null,
  PRIMARY KEY(snippet_id, viewer_ip, viewed_at)
);
