{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    nodejs_18
    nodePackages.npm
    openssl
    pkg-config
    prisma-engines
    postgresql  # Optional: for local development
  ];
  
  shellHook = ''
    # Set up Prisma engine paths for NixOS
    export PRISMA_QUERY_ENGINE_LIBRARY=${pkgs.prisma-engines}/lib/libquery_engine.node
    export PRISMA_QUERY_ENGINE_BINARY=${pkgs.prisma-engines}/bin/query-engine
    export PRISMA_SCHEMA_ENGINE_BINARY=${pkgs.prisma-engines}/bin/schema-engine
    export PRISMA_INTROSPECTION_ENGINE_BINARY=${pkgs.prisma-engines}/bin/introspection-engine
    export PRISMA_FMT_BINARY=${pkgs.prisma-engines}/bin/prisma-fmt
    
    # Skip checksum validation since we're providing our own engines
    export PRISMA_ENGINES_CHECKSUM_IGNORE_MISSING=1
    
    # For development
    export DATABASE_URL="postgresql://localhost:5432/language_operator_dev?schema=public"
    
    echo "🔧 NixOS Prisma environment configured!"
    echo "📍 Query Engine: $PRISMA_QUERY_ENGINE_BINARY"
    echo "📍 Schema Engine: $PRISMA_SCHEMA_ENGINE_BINARY"
    echo ""
    echo "💡 To build: npm run build"
    echo "💡 To develop: npm run dev"
  '';
}