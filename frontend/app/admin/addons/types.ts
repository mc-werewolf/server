export type AddonProperties = {
  id?: string;
  header?: {
    name?: string;
    description?: string;
    version?: {
      major?: number;
      minor?: number;
      patch?: number;
      prerelease?: string;
    };
  };
  tags?: string[];
};

export type AddonVersion = {
  id: string;
  tag_name: string;
  published_at: string;
  properties: AddonProperties | null;
  properties_error: string | null;
};

export type Addon = {
  id: string;
  github_owner: string;
  github_repo: string;
  versions: AddonVersion[];
};
