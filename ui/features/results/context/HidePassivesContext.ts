import { createContext, useContext } from 'react';

export const HidePassivesContext = createContext(true);

export const useHidePassives = (): boolean => useContext(HidePassivesContext);
