import { type FormEvent, useState } from "react";
import {
  alert,
  card,
  input,
  primaryButton,
  secondaryButton,
} from "../ui/styles";
import {
  signInErrorMessage,
  useCreateProfile,
  useProfiles,
  useSignIn,
} from "./useSession";

interface Profile {
  id: string;
  displayName: string;
}

export function SignedOut() {
  const profiles = useProfiles();
  const [selected, setSelected] = useState<Profile | null>(null);
  const [creating, setCreating] = useState(false);

  if (profiles.isPending) {
    return <p className="text-sm text-slate-500">Loading profiles…</p>;
  }
  if (profiles.isError) {
    return <p className={alert}>Couldn't load profiles.</p>;
  }
  if (selected) {
    return <PinForm profile={selected} onBack={() => setSelected(null)} />;
  }
  if (creating || profiles.data.length === 0) {
    return (
      <CreateProfileForm
        canCancel={profiles.data.length > 0}
        onDone={() => setCreating(false)}
      />
    );
  }
  return (
    <section aria-labelledby="who" className={card}>
      <h2 id="who" className="mb-3 text-lg font-semibold">
        Who's playing?
      </h2>
      <ul className="flex flex-col gap-2">
        {profiles.data.map((p) => (
          <li key={p.id}>
            <button
              type="button"
              onClick={() => setSelected(p)}
              className="w-full rounded-xl bg-slate-100 px-4 py-3 text-left font-medium hover:bg-teal-50 dark:bg-slate-800 dark:hover:bg-slate-700"
            >
              {p.displayName}
            </button>
          </li>
        ))}
      </ul>
      <button
        type="button"
        onClick={() => setCreating(true)}
        className={`${secondaryButton} mt-3`}
      >
        New profile
      </button>
    </section>
  );
}

function PinForm({
  profile,
  onBack,
}: {
  profile: Profile;
  onBack: () => void;
}) {
  const signIn = useSignIn();
  const [pin, setPin] = useState("");

  const submit = (e: FormEvent) => {
    e.preventDefault();
    signIn.mutate({ profileId: profile.id, pin });
  };

  return (
    <form onSubmit={submit} className={`${card} flex flex-col gap-3`}>
      <h2 className="text-lg font-semibold">
        Hi {profile.displayName}, enter your PIN
      </h2>
      <label className="flex flex-col gap-1 text-sm font-medium">
        PIN for {profile.displayName}
        <input
          className={input}
          type="password"
          inputMode="numeric"
          autoComplete="current-password"
          pattern="[0-9]{4,8}"
          minLength={4}
          maxLength={8}
          required
          value={pin}
          onChange={(e) => setPin(e.target.value)}
        />
      </label>
      {signIn.isError && (
        <p role="alert" className={alert}>
          {signInErrorMessage(signIn.error)}
        </p>
      )}
      <button
        type="submit"
        className={primaryButton}
        disabled={signIn.isPending}
      >
        Sign in
      </button>
      <button type="button" onClick={onBack} className={secondaryButton}>
        Not {profile.displayName}?
      </button>
    </form>
  );
}

function CreateProfileForm({
  canCancel,
  onDone,
}: {
  canCancel: boolean;
  onDone: () => void;
}) {
  const create = useCreateProfile();
  const [name, setName] = useState("");
  const [pin, setPin] = useState("");
  const [repeat, setRepeat] = useState("");
  const [mismatch, setMismatch] = useState(false);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (pin !== repeat) {
      setMismatch(true);
      return;
    }
    setMismatch(false);
    create.mutate({ displayName: name, pin }, { onSuccess: onDone });
  };

  const pinProps = {
    className: input,
    type: "password",
    inputMode: "numeric",
    autoComplete: "new-password",
    pattern: "[0-9]{4,8}",
    minLength: 4,
    maxLength: 8,
    required: true,
  } as const;

  return (
    <form onSubmit={submit} className={`${card} flex flex-col gap-3`}>
      <h2 className="text-lg font-semibold">New profile</h2>
      <p className="text-sm text-slate-600 dark:text-slate-400">
        Each trainer gets their own collection, rules and history, protected by
        a PIN.
      </p>
      <label className="flex flex-col gap-1 text-sm font-medium">
        Trainer name
        <input
          className={input}
          required
          maxLength={40}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </label>
      <label className="flex flex-col gap-1 text-sm font-medium">
        New PIN
        <input
          {...pinProps}
          value={pin}
          onChange={(e) => setPin(e.target.value)}
        />
      </label>
      <label className="flex flex-col gap-1 text-sm font-medium">
        Repeat PIN
        <input
          {...pinProps}
          value={repeat}
          onChange={(e) => setRepeat(e.target.value)}
        />
      </label>
      <p className="text-xs text-slate-500">4 to 8 digits.</p>
      {mismatch && (
        <p role="alert" className={alert}>
          The PINs don't match.
        </p>
      )}
      {create.isError && (
        <p role="alert" className={alert}>
          {create.error.message}
        </p>
      )}
      <button
        type="submit"
        className={primaryButton}
        disabled={create.isPending}
      >
        Create profile
      </button>
      {canCancel && (
        <button type="button" onClick={onDone} className={secondaryButton}>
          Cancel
        </button>
      )}
    </form>
  );
}
