-- M2: temas del semillero (grafo de conocimiento y temas del inicio).
ALTER TABLE semilleros ADD COLUMN IF NOT EXISTS temas JSONB NOT NULL DEFAULT '[]';
