declare module "next/navigation" {
  export function useParams<T = Record<string, string>>(): T;
}
