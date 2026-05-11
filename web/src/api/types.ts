export * from "./generated";

export type AuthUser = {
  id: string;
  email: string;
  primary_currency: string;
};

export type Profile = {
  user: AuthUser;
};
