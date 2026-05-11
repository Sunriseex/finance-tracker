import { readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import YAML from "yaml";

const openapiPath = resolve("../docs/openapi.yaml");
const outputPath = resolve("src/api/generated.ts");
const document = YAML.parse(readFileSync(openapiPath, "utf8"));
const schemas = document.components?.schemas ?? {};

const lines = [
  "// Generated from docs/openapi.yaml. Do not edit by hand.",
  "",
];

for (const [name, schema] of Object.entries(schemas)) {
  lines.push(`export type ${name} = ${schemaToType(schema)};`, "");
}

writeFileSync(outputPath, `${lines.join("\n")}\n`);

function schemaToType(schema) {
  if (!schema) {
    return "unknown";
  }
  if (schema.$ref) {
    return refName(schema.$ref);
  }
  if (schema.allOf) {
    return schema.allOf.map(schemaToType).join(" & ");
  }
  if (schema.oneOf || schema.anyOf) {
    return (schema.oneOf ?? schema.anyOf).map(schemaToType).join(" | ");
  }
  if (schema.enum) {
    return schema.enum.map((value) => JSON.stringify(value)).join(" | ");
  }

  let type;
  switch (schema.type) {
    case "array":
      type = `${schemaToType(schema.items)}[]`;
      break;
    case "boolean":
      type = "boolean";
      break;
    case "integer":
    case "number":
      type = "number";
      break;
    case "object":
      type = objectType(schema);
      break;
    case "string":
      type = "string";
      break;
    default:
      type = "unknown";
  }

  return schema.nullable ? `${type} | null` : type;
}

function objectType(schema) {
  if (schema.additionalProperties && !schema.properties) {
    const valueType = schema.additionalProperties === true ? "unknown" : schemaToType(schema.additionalProperties);
    return `Record<string, ${valueType}>`;
  }

  const required = new Set(schema.required ?? []);
  const properties = Object.entries(schema.properties ?? {});
  if (properties.length === 0) {
    return "Record<string, unknown>";
  }

  const body = properties.map(([name, property]) => {
    const optional = required.has(name) ? "" : "?";
    return `  ${JSON.stringify(name)}${optional}: ${schemaToType(property)};`;
  });
  return `{\n${body.join("\n")}\n}`;
}

function refName(ref) {
  return ref.split("/").at(-1);
}
