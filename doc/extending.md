# Extending

![Extension Points](images/extension-points.png)

## Add New Entity

1. Add GORM model in `model/entity/`
2. Add repository in `model/repository/`
3. Add API handler in `api/`
4. Register routes in `main.go`

## GraphQL Extensions

Register in `custom/` with `gqlregistry.Register` — see [graphql.md](graphql.md).

## Tailwind CSS

```bash
npm install -D tailwindcss@3
npx tailwindcss -i ./input.css -o ./assets/tailwind.min.css --minify --content './html/**/*.html'
```
