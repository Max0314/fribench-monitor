import { writeFile } from "node:fs/promises";

const placeholder = new URL(
  "../../internal/webui/dist/.gitkeep",
  import.meta.url,
);

await writeFile(placeholder, "");
